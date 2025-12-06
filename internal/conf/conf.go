package conf

import "time"

// 全局配置结构体（不变）
type GlobalConfig struct {
	App   AppConfig   `mapstructure:"app"`   // 应用基础配置（包含应用名称）
	Gin   GinConfig   `mapstructure:"gin"`   // Gin 引擎配置
	JWT   JWTConfig   `mapstructure:"jwt"`   // JWT 认证配置
	Email EmailConfig `mapstructure:"email"` // 邮件发送配置
	Login LoginConfig `mapstructure:"login"` // 登录账号配置
	Log   LogConfig   `mapstructure:"log"`
	DB    DBConfig    `mapstructure:"db"`
}

// AppConfig 应用基础配置（核心：包含应用名称）
type AppConfig struct {
	Name    string `mapstructure:"name"`    // 应用名称（从配置中读取）
	Env     string `mapstructure:"env"`     // 环境（dev/test/prod）
	Host    string `mapstructure:"host"`    // 监听主机
	Port    int    `mapstructure:"port"`    // 监听端口
	Version string `mapstructure:"version"` // 应用版本
}

// 其他结构体（GinConfig/JWTConfig/EmailConfig/LoginConfig/LoginUser）保持不变...
type GinConfig struct {
	Mode               string        `mapstructure:"mode"`                 // 运行模式（debug/release/test）
	ReadTimeout        time.Duration `mapstructure:"read_timeout"`         // 读取超时
	WriteTimeout       time.Duration `mapstructure:"write_timeout"`        // 写入超时
	IdleTimeout        time.Duration `mapstructure:"idle_timeout"`         // 空闲超时
	MaxMultipartMemory int64         `mapstructure:"max_multipart_memory"` // 最大上传内存
}

type JWTConfig struct {
	SecretKey     string        `mapstructure:"secret_key"`     // 密钥（必须保密）
	Issuer        string        `mapstructure:"issuer"`         // 签发者
	Audience      string        `mapstructure:"audience"`       // 受众
	ExpireHours   time.Duration `mapstructure:"expire_hours"`   // 过期时间（小时）
	RefreshHours  time.Duration `mapstructure:"refresh_hours"`  // 刷新令牌过期时间（小时）
	SigningMethod string        `mapstructure:"signing_method"` // 签名算法（HS256/HS512）
}

type EmailConfig struct {
	SMTPServer string `mapstructure:"smtp_server"` // SMTP服务器地址（如：smtp.qq.com:587）
	Username   string `mapstructure:"username"`    // 发件人邮箱账号
	Password   string `mapstructure:"password"`    // 授权码/密码
	From       string `mapstructure:"from"`        // 发件人显示邮箱（需与Username一致）
}

type LoginConfig struct {
	AdminUser     string      `mapstructure:"admin_user"`     // 管理员用户名
	AdminPassword string      `mapstructure:"admin_password"` // 管理员密码
	Users         []LoginUser `mapstructure:"users"`          // 多用户列表
}

type LoginUser struct {
	Username string `mapstructure:"username"` // 用户名
	Password string `mapstructure:"password"` // 密码
	Role     string `mapstructure:"role"`     // 角色
}

type LogConfig struct {
	LogDir      string `mapstructure:"log_dir"`         // 日志位置（目录）
	FileName    string `mapstructure:"log_file"`        // 日志文件名
	Level       string `mapstructure:"log_level"`       // 日志级别（debug/info/warn/error）
	MaxSize     int    `mapstructure:"log_max_size"`    // 单个文件最大MB
	MaxBackups  int    `mapstructure:"log_max_backups"` // 保留文件数
	MaxAge      int    `mapstructure:"log_max_age"`     // 保留天数
	Compress    bool   `mapstructure:"log_compress"`    // 是否压缩归档
	ColorEnable bool   `mapstructure:"log_color"`       // 是否开启彩色输出
	AddCaller   bool   `mapstructure:"log_add_caller"`  // 是否显示文件行号
}

// DBConfig 数据库配置（SQLite + GORM）
type DBConfig struct {
	SQLite SQLiteConfig `mapstructure:"sqlite"` // SQLite配置
	GORM   GORMConfig   `mapstructure:"gorm"`   // GORM配置
}

// SQLiteConfig SQLite专属配置
type SQLiteConfig struct {
	Path              string `mapstructure:"path"`                // 数据库文件路径（必填）
	AutoMigrate       bool   `mapstructure:"auto_migrate"`        // 自动迁移表结构（必填）
	DisableForeignKey bool   `mapstructure:"disable_foreign_key"` // 禁用外键（必填）
	JournalMode       string `mapstructure:"journal_mode"`        // 日志模式（默认WAL）
	CacheSize         int    `mapstructure:"cache_size"`          // 缓存大小（KB，默认2048）
	Synchronous       string `mapstructure:"synchronous"`         // 同步模式（默认FULL）
}

// GORMConfig GORM通用配置
type GORMConfig struct {
	LogMode         string        `mapstructure:"log_mode"`           // 日志级别（必填）
	SlowThreshold   time.Duration `mapstructure:"slow_threshold"`     // 慢查询阈值（必填）
	MaxOpenConns    int           `mapstructure:"max_open_conns"`     // 最大打开连接数（必填）
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`     // 最大空闲连接数（必填）
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`  // 连接最大存活时间（必填）
	ConnMaxIdleTime time.Duration `mapstructure:"conn_max_idle_time"` // 连接最大空闲时间（必填）
}
