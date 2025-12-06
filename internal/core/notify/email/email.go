package email

import (
	"context"
	"encoding/base64"
	"fmt"
	"health-probe/internal/conf"
	"io"
	"strings"
	"time"

	"github.com/emersion/go-sasl"
	"github.com/emersion/go-smtp"
	"go.uber.org/zap"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/encoding/traditionalchinese"
	"golang.org/x/text/transform"
)

// EmailConfig SMTP服务器配置（实例化时传入）
type EmailConfig struct {
	SMTPServer string // SMTP服务器地址（如：smtp.qq.com:587）
	Username   string // 发件人邮箱账号（如：xxx@qq.com）
	Password   string // 发件人授权码/密码
	From       string // 发件人显示邮箱（需与Username一致）
}

// EmailSender 邮件发送接口
type Sender interface {
	Send(ctx context.Context, req *EmailSendRequest)
}

// EmailSendRequest 邮件发送请求（支持HTML/纯文本）
type EmailSendRequest struct {
	To      []string // 收件人邮箱列表（必填）
	Subject string   // 邮件主题（必填）
	Body    string   // 邮件内容（必填）
	IsHTML  bool     // 是否为HTML格式（默认false，纯文本）
	Charset string   // 字符编码（默认utf-8）
}

// EmailSendResponse 邮件发送响应
type EmailSendResponse struct {
	Success bool   // 是否发送成功
	Message string // 响应信息（成功提示或错误详情）
}

// emailSender 实现类（持有配置）
type emailSender struct {
	config *conf.EmailConfig
	logger *zap.Logger
}

// NewEmailSender 创建邮件发送实例（加载配置）
func NewSender(config *conf.EmailConfig, logger *zap.Logger) (Sender, error) {
	// 校验配置合法性
	if err := validateConfig(config); err != nil {
		return nil, fmt.Errorf("invalid config: %v", err)
	}

	return &emailSender{
		config: config,
		logger: logger,
	}, nil
}

// Send 执行邮件发送（支持HTML/纯文本）
func (s *emailSender) Send(ctx context.Context, req *EmailSendRequest) {
	// 校验请求参数并设置默认值
	if err := s.validateAndSetDefaults(req); err != nil {
		s.logger.Error("invalid request: %v", zap.Error(err))
		return
	}

	// 异步发送邮件
	go func(req *EmailSendRequest, s *emailSender) {
		// 构建邮件内容
		emailContent := s.buildEmailContent(req)

		// 配置重试参数
		maxRetries := 2
		retryDelay := 3 * time.Second
		var finalErr error

		for attempt := 0; attempt <= maxRetries; attempt++ {
			// 智能选择连接方式
			var client *smtp.Client
			var err error

			// 检查服务器端口，选择适当的连接方式
			if strings.HasSuffix(s.config.SMTPServer, ":465") {
				// 使用TLS连接（SMTPS）
				client, err = smtp.DialTLS(s.config.SMTPServer, nil)
				if err != nil {
					s.logger.Warn(
						"TLS connection failed, trying plain connection",
						zap.String("server", s.config.SMTPServer),zap.Error(err),
					)
					// 回退到普通连接
					client, err = smtp.Dial(s.config.SMTPServer)
				}
			} else if strings.HasSuffix(s.config.SMTPServer, ":587") ||
				strings.HasSuffix(s.config.SMTPServer, ":25") {
				// 尝试使用STARTTLS（587或25端口）
				client, err = smtp.DialStartTLS(s.config.SMTPServer, nil)
				if err != nil {
					s.logger.Warn(
						"STARTTLS connection failed, trying plain connection",
						zap.String("server", s.config.SMTPServer),
						zap.Error(err),
					)
					// 回退到普通连接
					client, err = smtp.Dial(s.config.SMTPServer)
				}
			} else {
				// 默认使用普通连接
				client, err = smtp.Dial(s.config.SMTPServer)
			}

			if err != nil {
				finalErr = err
				s.logger.Error(
					"failed to connect to SMTP server",
					zap.String("server", s.config.SMTPServer),
					zap.Error(err),
				)
				if attempt < maxRetries {
					time.Sleep(retryDelay)
					continue
				}
				return
			}
			defer client.Close()

			// 认证
			auth := sasl.NewPlainClient("", s.config.Username, s.config.Password)
			if err := client.Auth(auth); err != nil {
				finalErr = err
				s.logger.Error(
					"SMTP authentication failed",
					zap.String("username", s.config.Username),
					zap.Error(err),
				)
				if attempt < maxRetries {
					time.Sleep(retryDelay)
					continue
				}
				return
			}

			// 发送邮件
			err = client.SendMail(s.config.From, req.To, strings.NewReader(emailContent))

			if err == nil {
				s.logger.Info(
					"email sent successfully",
					zap.String("from", s.config.From),
					zap.String("to", strings.Join(req.To, ",")),
					zap.Int("attempt", attempt+1),
				)
				return
			}

			// 处理发送错误
			finalErr = err
			s.logger.Warn(
				"email send attempt failed",
				zap.String("from", s.config.From),
				zap.String("to", strings.Join(req.To, ",")),
				zap.Int("attempt", attempt+1),
				zap.Int("max_retries", maxRetries),
				zap.Error(err),
			)

			// 分析错误类型
			switch err := err.(type) {
			case *smtp.SMTPError:
				switch {
				case err.Code >= 500 && err.Code < 600:
					s.logger.Error(
						"email send failed permanently (SMTP 5xx error)",
						zap.String("from", s.config.From),
						zap.String("to", strings.Join(req.To, ",")),
						zap.Int("smtp_code", err.Code),
						zap.String("smtp_msg", err.Message),
						zap.Error(err),
					)
					return

				case err.Code >= 400 && err.Code < 500:
					if attempt >= maxRetries {
						s.logger.Error(
							"email send failed after retries (SMTP 4xx temporary error)",
							zap.String("from", s.config.From),
							zap.String("to", strings.Join(req.To, ",")),
							zap.Int("smtp_code", err.Code),
							zap.String("smtp_msg", err.Message),
							zap.Int("total_attempts", attempt+1),
							zap.Error(err),
						)
						return
					}
					time.Sleep(retryDelay)
					continue

				default:
					s.logger.Error(
						"email send failed due to SMTP error",
						zap.String("from", s.config.From),
						zap.String("to", strings.Join(req.To, ",")),
						zap.Int("smtp_code", err.Code),
						zap.String("smtp_msg", err.Message),
						zap.Error(err),
					)
					return
				}

			default:
				s.logger.Error(
					"email send failed due to system/network error",
					zap.String("from", s.config.From),
					zap.String("to", strings.Join(req.To, ",")),
					zap.Error(err),
				)
				return
			}
		}

		// 所有重试失败
		s.logger.Error(
			"email send failed after all retries",
			zap.String("from", s.config.From),
			zap.String("to", strings.Join(req.To, ",")),
			zap.Error(finalErr),
		)
	}(req, s)
}

// 在文件末尾添加以下方法

// tryAlternativeAuth 尝试其他认证机制
func (s *emailSender) tryAlternativeAuth() sasl.Client {
	// 尝试LOGIN认证（某些服务器支持）
	return sasl.NewLoginClient(s.config.Username, s.config.Password)
}

// isAuthError 判断是否为认证相关错误
func (s *emailSender) isAuthError(err error) bool {
	if smtpErr, ok := err.(*smtp.SMTPError); ok {
		// 535: Authentication failed
		// 530: Authentication required
		return smtpErr.Code == 535 || smtpErr.Code == 530
	}
	return false
}

// validateConfig 校验SMTP配置
func validateConfig(config *conf.EmailConfig) error {
	if config == nil {
		return fmt.Errorf("config is nil")
	}
	if config.SMTPServer == "" {
		return fmt.Errorf("smtp server is required")
	}
	if config.Username == "" {
		return fmt.Errorf("username is required")
	}
	if config.Password == "" {
		return fmt.Errorf("password is required")
	}
	if config.From == "" {
		return fmt.Errorf("from email is required")
	}
	return nil
}

// validateAndSetDefaults 校验请求参数并设置默认值
func (s *emailSender) validateAndSetDefaults(req *EmailSendRequest) error {
	if req == nil {
		return fmt.Errorf("request is nil")
	}
	if len(req.To) == 0 {
		return fmt.Errorf("recipient list is empty")
	}
	if req.Subject == "" {
		return fmt.Errorf("subject is required")
	}
	if req.Body == "" {
		return fmt.Errorf("body is required")
	}
	// 默认字符编码：utf-8
	if req.Charset == "" {
		req.Charset = "utf-8"
	}
	return nil
}

// buildEmailContent 构建邮件内容（支持HTML/纯文本）
func (s *emailSender) buildEmailContent(req *EmailSendRequest) string {
	var builder strings.Builder

	// 邮件头
	builder.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(req.To, ",")))
	builder.WriteString(fmt.Sprintf("From: %s\r\n", s.config.From))
	builder.WriteString(fmt.Sprintf("Subject: =?%s?B?%s?=\r\n", req.Charset, base64Encode(req.Subject, req.Charset))) // 主题Base64编码（支持中文）
	// 内容类型（根据IsHTML切换）
	if req.IsHTML {
		builder.WriteString(fmt.Sprintf("Content-Type: text/html; charset=%s\r\n", req.Charset))
	} else {
		builder.WriteString(fmt.Sprintf("Content-Type: text/plain; charset=%s\r\n", req.Charset))
	}
	builder.WriteString("MIME-Version: 1.0\r\n") // 声明MIME版本
	builder.WriteString("\r\n")                  // 空行分隔头和正文
	// 邮件正文
	builder.WriteString(req.Body)

	return builder.String()
}

// getEncoding 根据字符集名称获取对应的编码
func getEncoding(charset string) (encoding.Encoding, error) {
	switch strings.ToLower(charset) {
	case "utf-8", "utf8":
		return encoding.Nop, nil
	case "gbk":
		return simplifiedchinese.GBK, nil
	case "gb18030":
		return simplifiedchinese.GB18030, nil
	case "big5":
		return traditionalchinese.Big5, nil
	case "iso-8859-1", "latin1":
		return charmap.ISO8859_1, nil
	case "iso-8859-2", "latin2":
		return charmap.ISO8859_2, nil
	case "windows-1252":
		return charmap.Windows1252, nil
	default:
		return nil, fmt.Errorf("unsupported charset: %s", charset)
	}
}

// base64Encode Base64编码（处理中文主题）
func base64Encode(str, charset string) string {
	// 如果是UTF-8编码，直接进行Base64编码
	if strings.ToLower(charset) == "utf-8" || strings.ToLower(charset) == "utf8" {
		return base64.StdEncoding.EncodeToString([]byte(str))
	}

	// 获取目标编码
	enc, err := getEncoding(charset)
	if err != nil {
		// 不支持的编码，直接使用原字符串进行Base64编码
		return base64.StdEncoding.EncodeToString([]byte(str))
	}

	// 将UTF-8字符串转换为目标编码
	transformer := enc.NewEncoder()
	reader := transform.NewReader(strings.NewReader(str), transformer)
	data, err := io.ReadAll(reader)
	if err != nil {
		// 转换失败则使用原字符串
		return base64.StdEncoding.EncodeToString([]byte(str))
	}

	// Base64编码
	return base64.StdEncoding.EncodeToString(data)
}
