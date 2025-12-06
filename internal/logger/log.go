package logger

import (
	"context"
	"fmt"
	"health-probe/internal/conf"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// 全局日志实例（对外暴露）
var Logger *zap.Logger
var SugaredLogger *zap.SugaredLogger

// ------------------------------
// 对接你的配置：日志相关配置结构体
// 注意：如果你的现有配置中日志字段名不同，修改这里的字段名即可（保持JSON标签一致）
// ------------------------------
type LogConfig struct {
	LogDir      string `json:"log_dir" yaml:"log_dir"`                 // 日志位置（目录）
	FileName    string `json:"log_file" yaml:"log_file"`               // 日志文件名
	Level       string `json:"log_level" yaml:"log_level"`             // 日志级别（debug/info/warn/error）
	MaxSize     int    `json:"log_max_size" yaml:"log_max_size"`       // 单个文件最大MB
	MaxBackups  int    `json:"log_max_backups" yaml:"log_max_backups"` // 保留文件数
	MaxAge      int    `json:"log_max_age" yaml:"log_max_age"`         // 保留天数
	Compress    bool   `json:"log_compress" yaml:"log_compress"`       // 是否压缩归档
	ColorEnable bool   `json:"log_color" yaml:"log_color"`             // 是否开启彩色输出
	AddCaller   bool   `json:"log_add_caller" yaml:"log_add_caller"`   // 是否显示文件行号
}

// ------------------------------
// 默认配置（当你的配置中某些字段未设置时兜底）
// ------------------------------
func getDefaultLogConfig() LogConfig {
	return LogConfig{
		LogDir:      "./logs",  // 默认日志目录
		FileName:    "app.log", // 默认日志文件名
		Level:       "info",    // 默认级别：info
		MaxSize:     100,       // 默认单个文件100MB
		MaxBackups:  30,        // 默认保留30个文件
		MaxAge:      7,         // 默认保留7天
		Compress:    true,      // 默认压缩归档
		ColorEnable: false,     // 默认不开启彩色（生产环境）
		AddCaller:   true,      // 默认显示文件行号
	}
}

// ------------------------------
// 核心初始化方法：从你的配置中加载日志配置
// ------------------------------
// InitLogger 初始化日志器
// 参数：yourConfig 你的全局配置结构体（必须包含 LogConfig 对应的字段）
// 用法：logger.InitLogger(globalConfig.Log) （假设你的全局配置中日志字段叫 Log）
func InitLogger(conf conf.LogConfig) error {
	// 1. 合并配置：你的配置 + 默认配置（未设置的字段用默认值）
	cfg := mergeConfig(conf)

	// 2. 验证并转换日志级别（字符串转zapcore.Level）
	logLevel, err := parseLogLevel(cfg.Level)
	if err != nil {
		return fmt.Errorf("invalid log level: %w", err)
	}

	// 3. 创建日志目录（不存在则自动创建）
	if err := os.MkdirAll(cfg.LogDir, 0755); err != nil {
		return fmt.Errorf("create log dir failed: %w", err)
	}

	// 4. 配置日志轮转（文件切割）
	logFilePath := filepath.Join(cfg.LogDir, cfg.FileName)
	fileWriter := &lumberjack.Logger{
		Filename:   logFilePath,
		MaxSize:    cfg.MaxSize,
		MaxBackups: cfg.MaxBackups,
		MaxAge:     cfg.MaxAge,
		Compress:   cfg.Compress,
		LocalTime:  true, // 归档文件名使用本地时间
	}

	// 5. 配置日志编码器（结构化输出格式）
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalLevelEncoder, // 级别大写（INFO/WARN/ERROR）
		EncodeTime:     zapcore.ISO8601TimeEncoder,  // 时间格式：2024-05-20T15:04:05.123Z
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder, // 调用者格式：pkg/file.go:123
		FunctionKey:    zapcore.OmitKey,            // 不显示函数名（减少日志体积）
	}

	// 6. 彩色输出配置（根据你的 ColorEnable 字段控制）
	var encoder zapcore.Encoder
	if cfg.ColorEnable {
		encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder // 彩色级别
		encoder = zapcore.NewConsoleEncoder(encoderConfig)           // 控制台友好格式
	} else {
		encoder = zapcore.NewJSONEncoder(encoderConfig) // JSON格式（便于日志收集）
	}

	// 7. 配置输出目标（文件 + 控制台可选）
	var core zapcore.Core
	// 彩色输出时默认同时输出到控制台和文件；否则只输出到文件（生产模式）
	if cfg.ColorEnable {
		core = zapcore.NewTee(
			zapcore.NewCore(encoder, zapcore.AddSync(os.Stdout), logLevel),                                // 控制台
			zapcore.NewCore(zapcore.NewJSONEncoder(encoderConfig), zapcore.AddSync(fileWriter), logLevel), // 文件（JSON）
		)
	} else {
		core = zapcore.NewCore(encoder, zapcore.AddSync(fileWriter), logLevel) // 仅文件
	}

	// 8. 构建日志器（添加调用者、堆栈跟踪）
	loggerBuilder := zap.New(core)
	if cfg.AddCaller {
		loggerBuilder = loggerBuilder.WithOptions(zap.WithCaller(true), zap.AddCallerSkip(1)) // 显示文件行号
	}
	loggerBuilder = loggerBuilder.WithOptions(zap.AddStacktrace(zapcore.ErrorLevel)) // ERROR级别自动加堆栈

	// 9. 初始化全局实例
	Logger = loggerBuilder
	SugaredLogger = Logger.Sugar()

	// 初始化成功日志（仅输出到文件，避免控制台刷屏）
	// Logger.Info("logger initialized successfully",
	// 	zap.String("log_path", logFilePath),
	// 	zap.String("level", cfg.Level),
	// 	zap.Bool("color_enable", cfg.ColorEnable),
	// 	zap.Bool("add_caller", cfg.AddCaller),
	// )

	// 程序退出时刷新日志缓冲区
	runtime.SetFinalizer(&Logger, func(l **zap.Logger) {
		_ = (*l).Sync()
	})

	return nil
}

// ------------------------------
// 辅助函数：合并配置（你的配置覆盖默认配置）
// ------------------------------
func mergeConfig(cfg conf.LogConfig) LogConfig {
	defaultCfg := getDefaultLogConfig()

	// 日志目录：你的配置有值则用你的，否则用默认
	if cfg.LogDir != "" {
		defaultCfg.LogDir = cfg.LogDir
	}
	// 日志文件名
	if cfg.FileName != "" {
		defaultCfg.FileName = cfg.FileName
	}
	// 日志级别（不区分大小写）
	if strings.TrimSpace(cfg.Level) != "" {
		defaultCfg.Level = strings.ToLower(cfg.Level)
	}
	// 单个文件最大大小（必须大于0）
	if cfg.MaxSize > 0 {
		defaultCfg.MaxSize = cfg.MaxSize
	}
	// 保留文件数（必须大于0）
	if cfg.MaxBackups > 0 {
		defaultCfg.MaxBackups = cfg.MaxBackups
	}
	// 保留天数（必须大于0）
	if cfg.MaxAge > 0 {
		defaultCfg.MaxAge = cfg.MaxAge
	}
	// 压缩归档（你的配置明确设置为false则禁用）
	defaultCfg.Compress = cfg.Compress
	// 彩色输出（你的配置明确设置为true则开启）
	defaultCfg.ColorEnable = cfg.ColorEnable
	// 显示调用者（你的配置明确设置为false则禁用）
	defaultCfg.AddCaller = cfg.AddCaller

	return defaultCfg
}

// ------------------------------
// 辅助函数：字符串级别转zapcore.Level
// ------------------------------
func parseLogLevel(levelStr string) (zapcore.Level, error) {
	var level zapcore.Level
	switch strings.ToLower(levelStr) {
	case "debug":
		level = zapcore.DebugLevel
	case "info":
		level = zapcore.InfoLevel
	case "warn":
		level = zapcore.WarnLevel
	case "error":
		level = zapcore.ErrorLevel
	case "dpanic":
		level = zapcore.DPanicLevel
	case "panic":
		level = zapcore.PanicLevel
	case "fatal":
		level = zapcore.FatalLevel
	default:
		return zapcore.InfoLevel, fmt.Errorf("%s (supported: debug/info/warn/error/dpanic/panic/fatal)", levelStr)
	}
	return level, nil
}

// ------------------------------
// 上下文追踪（可选，用于分布式链路追踪）
// ------------------------------
type ctxKey string

const TraceIDKey ctxKey = "trace_id"

// WithTraceID 给上下文添加traceID
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, TraceIDKey, traceID)
}

// CtxLogger 从上下文获取带traceID的日志器
func CtxLogger(ctx context.Context) *zap.Logger {
	if traceID, ok := ctx.Value(TraceIDKey).(string); ok && traceID != "" && Logger != nil {
		return Logger.With(zap.String("trace_id", traceID))
	}
	return Logger
}

// CtxSugaredLogger 带上下文的SugaredLogger
func CtxSugaredLogger(ctx context.Context) *zap.SugaredLogger {
	return CtxLogger(ctx).Sugar()
}

// ------------------------------
// 简化日志方法（直接调用全局实例，避免每次写 Logger.Xxx）
// ------------------------------
func Debug(msg string, fields ...zap.Field) {
	if Logger != nil {
		Logger.Debug(msg, fields...)
	}
}

func Debugf(tpl string, args ...interface{}) {
	if SugaredLogger != nil {
		SugaredLogger.Debugf(tpl, args...)
	}
}

func Info(msg string, fields ...zap.Field) {
	if Logger != nil {
		Logger.Info(msg, fields...)
	}
}

func Infof(tpl string, args ...interface{}) {
	if SugaredLogger != nil {
		SugaredLogger.Infof(tpl, args...)
	}
}

func Warn(msg string, fields ...zap.Field) {
	if Logger != nil {
		Logger.Warn(msg, fields...)
	}
}

func Warnf(tpl string, args ...interface{}) {
	if SugaredLogger != nil {
		SugaredLogger.Warnf(tpl, args...)
	}
}

func Error(msg string, fields ...zap.Field) {
	if Logger != nil {
		Logger.Error(msg, fields...)
	}
}

func Errorf(tpl string, args ...interface{}) {
	if SugaredLogger != nil {
		SugaredLogger.Errorf(tpl, args...)
	}
}

func Fatal(msg string, fields ...zap.Field) {
	if Logger != nil {
		Logger.Fatal(msg, fields...)
	}
}

func Fatalf(tpl string, args ...interface{}) {
	if SugaredLogger != nil {
		SugaredLogger.Fatalf(tpl, args...)
	}
}

// Sync 刷新日志缓冲区（程序退出时调用）
func Sync() error {
	if Logger != nil {
		return Logger.Sync()
	}
	return nil
}
