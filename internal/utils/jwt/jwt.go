package jwt

import (
	"errors"
	"time"

	"health-probe/internal/conf"

	"github.com/golang-jwt/jwt/v5"
)

// 自定义 Claims（包含标准 Claims + 业务字段）
// 可根据业务需求添加额外字段（如 UserID、Username、Role 等）
type CustomClaims struct {
	jwt.RegisteredClaims        // 嵌入标准注册 Claims（Issuer、ExpiresAt、Audience 等）
	UserID               string `json:"user_id"`  // 示例：用户 ID（业务字段）
	Username             string `json:"username"` // 示例：用户名（业务字段）
}

// JwtService 封装 JWT 操作（依赖配置）
type JwtService struct {
	config *conf.JWTConfig
}

// NewJWTService 初始化 JWT 服务
func NewJWTService(cfg *conf.JWTConfig) *JwtService {
	// 校验配置合法性（HS256 要求密钥至少 32 字节）
	if cfg.SigningMethod == "HS256" && len(cfg.SecretKey) < 32 {
		panic("HS256 算法要求 SecretKey 至少 32 字节")
	}
	return &JwtService{config: cfg}
}

// --------------- 核心功能 1：生成 Access Token ---------------
// GenerateAccessToken 生成访问令牌（短期有效，默认 2 小时）
func (s *JwtService) GenerateAccessToken(userID, username string) (string, error) {
	// 1. 构造自定义 Claims
	claims := CustomClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.config.Issuer,                                                                     // 签发者
			Audience:  jwt.ClaimStrings{s.config.Audience},                                                 // 受众（数组类型）
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * time.Duration(s.config.ExpireHours))), // 过期时间
			IssuedAt:  jwt.NewNumericDate(time.Now()),                                                      // 签发时间
			NotBefore: jwt.NewNumericDate(time.Now()),                                                      // 生效时间（立即生效）
		},
		UserID:   userID,   // 业务字段：用户 ID
		Username: username, // 业务字段：用户名
	}

	// 2. 选择签名算法（此处固定为 HS256，与配置一致）
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// 3. 用密钥签名并生成 Token 字符串
	return token.SignedString([]byte(s.config.SecretKey))
}

// --------------- 核心功能 2：生成 Refresh Token ---------------
// GenerateRefreshToken 生成刷新令牌（长期有效，默认 24 小时）
// 用途：Access Token 过期后，用 Refresh Token 免登录刷新新的 Access Token
func (s *JwtService) GenerateRefreshToken(userID string) (string, error) {
	// Refresh Token 无需携带过多业务字段，仅需用户标识即可
	claims := jwt.RegisteredClaims{
		Issuer:    s.config.Issuer,
		Audience:  jwt.ClaimStrings{s.config.Audience},
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * time.Duration(s.config.RefreshHours))),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		Subject:   userID, // 用 Subject 存储用户 ID（简化 Claims）
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.config.SecretKey))
}

// --------------- 核心功能 3：验证 Access Token ---------------
// ValidateAccessToken 验证访问令牌的合法性（签名、过期时间、签发者、受众）
// 返回解析后的 CustomClaims，供业务使用
func (s *JwtService) ValidateAccessToken(tokenStr string) (*CustomClaims, error) {
	// 1. 定义验证函数（校验签名 + 标准 Claims）
	token, err := jwt.ParseWithClaims(
		tokenStr,
		&CustomClaims{}, // 目标 Claims 类型
		func(token *jwt.Token) (interface{}, error) {
			// 校验签名算法是否为配置的 HS256
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, errors.New("不支持的签名算法")
			}
			// 返回签名密钥
			return []byte(s.config.SecretKey), nil
		},
		// 强制校验标准 Claims（Issuer、Audience、ExpiresAt）
		jwt.WithIssuer(s.config.Issuer),
		jwt.WithAudience(s.config.Audience),
		jwt.WithExpirationRequired(),
	)

	// 2. 处理解析错误（签名无效、过期、Claims 不匹配等）
	if err != nil {
		return nil, wrapJWTError(err)
	}

	// 3. 提取自定义 Claims
	claims, ok := token.Claims.(*CustomClaims)
	if !ok {
		return nil, errors.New("Claims格式错误")
	}

	return claims, nil
}

// --------------- 核心功能 4：验证 Refresh Token ---------------
// ValidateRefreshToken 验证刷新令牌的合法性
// 返回用户 ID（用于生成新的 Access Token）
func (s *JwtService) ValidateRefreshToken(tokenStr string) (string, error) {
	token, err := jwt.ParseWithClaims(
		tokenStr,
		&jwt.RegisteredClaims{},
		func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, errors.New("不支持的签名算法")
			}
			return []byte(s.config.SecretKey), nil
		},
		jwt.WithIssuer(s.config.Issuer),
		jwt.WithAudience(s.config.Audience),
		jwt.WithExpirationRequired(),
	)

	if err != nil {
		return "", wrapJWTError(err)
	}

	claims, ok := token.Claims.(*jwt.RegisteredClaims)
	if !ok {
		return "", errors.New("Claims格式错误")
	}

	// 返回存储在 Subject 中的用户 ID
	return claims.Subject, nil
}

// --------------- 核心功能 5：刷新 Access Token ---------------
// RefreshAccessToken 通过合法的 Refresh Token 生成新的 Access Token
func (s *JwtService) RefreshAccessToken(refreshTokenStr string) (newAccessToken string, err error) {
	// 1. 验证 Refresh Token
	userID, err := s.ValidateRefreshToken(refreshTokenStr)
	if err != nil {
		return "", err
	}

	// 2. 生成新的 Access Token（此处 Username 可从数据库查询，示例用空字符串）
	// 实际业务中，建议从用户中心查询用户完整信息（如 Username、Role 等）
	return s.GenerateAccessToken(userID, "")
}

// --------------- 辅助函数：统一错误处理 ---------------
// wrapJWTError 将 jwt 库的错误转为易读的业务错误
func wrapJWTError(err error) error {
	switch {
	case errors.Is(err, jwt.ErrTokenExpired):
		return errors.New("token 已过期")
	case errors.Is(err, jwt.ErrTokenInvalidClaims):
		return errors.New("token claims 无效")
	case errors.Is(err, jwt.ErrTokenSignatureInvalid):
		return errors.New("token 签名无效")
	case errors.Is(err, jwt.ErrTokenMalformed):
		return errors.New("token 格式错误")
	default:
		return errors.New("token 验证失败：" + err.Error())
	}
}
