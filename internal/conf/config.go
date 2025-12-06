package conf

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

// 全局变量
var (
	Cfg       GlobalConfig // 全局配置实例
	cfgViper  *viper.Viper // viper 实例（用于热加载）
	mu        sync.RWMutex // 读写锁（保证热加载线程安全）
	tempViper *viper.Viper // 临时 viper 实例（用于初始获取应用名）
)

// LoadConfig 加载配置（支持：双路径 + 从配置取应用名 + 热加载）
func Loaded() GlobalConfig {
	// 步骤1：初始化临时 viper，获取初始应用名（用于拼接 /etc/应用名/ 路径）
	if err := initTempViper(); err != nil {
		fmt.Printf("init temp viper failed: %v", err)
		os.Exit(1)
	}

	// 步骤2：获取应用名（优先从配置文件，其次默认值）
	appName := tempViper.GetString("app.name")
	if appName == "" {
		appName = "health-probe" // 最终默认值（防止配置缺失）
	}
	// fmt.Printf("detected app name: %s\n", appName)

	// 步骤3：初始化主 viper，支持双路径 + 热加载
	cfgViper = viper.New()
	setDefaultConfig(cfgViper) // 设置默认值

	// 配置文件路径（优先级：应用根目录 > /etc/应用名/）
	cfgViper.AddConfigPath(".")         // 移除重复的 AddConfigPath
	cfgViper.AddConfigPath("./configs") // 明确添加 configs 目录（原代码写了两次 "."，这里修正为实际需求）
	// 路径2：/etc/应用名/（动态拼接应用名）
	etcPath := filepath.Join("/etc", appName)
	cfgViper.AddConfigPath(etcPath)

	// 配置文件名称和类型
	cfgViper.SetConfigName("config")
	cfgViper.SetConfigType("yaml")

	// 环境变量配置（优先级：环境变量 > 配置文件 > 默认值）
	cfgViper.AutomaticEnv()
	cfgViper.SetEnvPrefix("HEALTH_PROBE")
	cfgViper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// 步骤4：读取完整配置
	configFound := true
	if err := cfgViper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			configFound = false
			fmt.Println("warning: no config file found in any path, using env vars or defaults")
		} else {
			fmt.Printf("read config failed: %v", err)
			os.Exit(1)
		}
	}

	// 步骤5：解析并校验配置
	if err := unmarshalAndValidate(); err != nil {
		fmt.Printf("init config failed: %v", err)
		os.Exit(1)
	}

	// 步骤6：仅当未找到任何配置文件时，生成默认配置（关键调整）
	if !configFound {
		if err := SaveDefaultConfig(appName); err != nil {
			fmt.Printf("warning: failed to generate default config: %v\n", err)
		}
	}

	// 步骤7：启动热加载
	startConfigWatch()

	// fmt.Printf("config loaded successfully (app: %s, env: %s, hot-reload enabled)\n", Cfg.App.Name, Cfg.App.Env)
	return Cfg
}

// initTempViper 初始化临时 viper，用于获取应用名（仅读取必要路径）
func initTempViper() error {
	tempViper = viper.New()

	// 仅加载可能存储应用名的路径（不加载 /etc/，因为应用名未知）
	tempViper.AddConfigPath(".")
	tempViper.AddConfigPath(".")
	tempViper.SetConfigName("config")
	tempViper.SetConfigType("yaml")

	// 环境变量支持（用于覆盖应用名）
	tempViper.AutomaticEnv()
	tempViper.SetEnvPrefix("APP")
	// 使用 strings.Replacer 来替换环境变量中的分隔符
	tempViper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// 设置应用名默认值
	tempViper.SetDefault("app.name", "health-probe")

	// 尝试读取配置（忽略文件不存在错误）
	if err := tempViper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return fmt.Errorf("read temp config failed: %v", err)
		}
	}

	return nil
}

// setDefaultConfig 设置配置默认值（不变）
func setDefaultConfig(v *viper.Viper) {
	// App 配置
	v.SetDefault("app.env", "dev")
	v.SetDefault("app.host", "0.0.0.0")
	v.SetDefault("app.port", 8080)
	v.SetDefault("app.version", "v1.0.0")

	// Gin 配置
	v.SetDefault("gin.mode", "debug")
	v.SetDefault("gin.read_timeout", "5s")
	v.SetDefault("gin.write_timeout", "10s")
	v.SetDefault("gin.idle_timeout", "15s")
	v.SetDefault("gin.max_multipart_memory", 10<<20) // 10MB

	// JWT 配置
	v.SetDefault("jwt.secret_key", "default-secret-key-32bytes-long-1234")
	v.SetDefault("jwt.issuer", "health-probe")
	v.SetDefault("jwt.audience", "api-users")
	v.SetDefault("jwt.expire_hours", 2)
	v.SetDefault("jwt.refresh_hours", 24)
	v.SetDefault("jwt.signing_method", "HS256")

	// Email 配置
	v.SetDefault("email.smtp_server", "smtp.qq.com:587")
	v.SetDefault("email.from", "www.example.com@qq.com")
	v.SetDefault("email.username", "www.example.com@qq.com")
	v.SetDefault("email.password", "testdafdasfgagsdasdgasdg")

	// Login 配置
	v.SetDefault("login.admin_user", "admin")
	v.SetDefault("login.admin_password", "admin123")
	v.SetDefault("login.users", []LoginUser{})

	// Log 配置
	v.SetDefault("log.log_dir", "/var/logs/health-probe/")
	v.SetDefault("log.log_file", "health-probe.log")
	v.SetDefault("log.log_level", "info")
	v.SetDefault("log.log_max_size", 100)
	v.SetDefault("log.log_max_backups", 3)
	v.SetDefault("log.log_max_age", 7)
	v.SetDefault("log.log_compress", false)
	v.SetDefault("log.log_color", true)
	v.SetDefault("log.log_add_caller", true)

	// DB 配置
	// SQLite
	v.SetDefault("db.sqlite.path", "./health-probe.db") // 数据库文件路径（相对路径，自动创建data目录）
	v.SetDefault("db.sqlite.auto_migrate", true)        // 开启自动迁移（首次启动创建表）
	v.SetDefault("db.sqlite.disable_foreign_key", true) // 禁用外键（SQLite建议）
	v.SetDefault("db.sqlite.journal_mode", "WAL")       // 启用WAL模式（支持读写并发）
	v.SetDefault("db.sqlite.cache_size", 4096)          // 缓存4MB（提升查询性能）
	v.SetDefault("db.sqlite.synchronous", "FULL")       // 完全同步（保证数据持久化）
	// GORM
	v.SetDefault("db.gorm.log_mode", "info")          // 开发环境：info（输出SQL）；生产环境：silent
	v.SetDefault("db.gorm.slow_threshold", "200ms")   // 慢查询阈值（超过200ms记录警告）
	v.SetDefault("db.gorm.max_open_conns", 5)         // SQLite最大打开连接数（避免锁冲突）
	v.SetDefault("db.gorm.max_idle_conns", 3)         // 最大空闲连接数（≤max_open_conns）
	v.SetDefault("db.gorm.conn_max_lifetime", "1h")   // 连接最长存活1小时
	v.SetDefault("db.gorm.conn_max_idle_time", "10m") // 空闲连接10分钟后关闭

}

// unmarshalAndValidate 解析配置并校验合法性（加读写锁）
func unmarshalAndValidate() error {
	mu.Lock()
	defer mu.Unlock()

	// 解析配置到全局变量
	if err := cfgViper.Unmarshal(&Cfg); err != nil {
		return fmt.Errorf("unmarshal config failed: %v", err)
	}

	// 校验配置合法性
	return validateConfig(&Cfg)
}

// validateConfig 校验配置合法性（不变）
func validateConfig(cfg *GlobalConfig) error {
	// App 配置校验
	if cfg.App.Name == "" {
		cfg.App.Name = "health-probe"
	}
	if cfg.App.Port <= 0 || cfg.App.Port > 65535 {
		return fmt.Errorf("app.port must be between 1-65535")
	}

	// Gin 配置校验
	switch cfg.Gin.Mode {
	case "debug", "release", "test":
	default:
		return fmt.Errorf("gin.mode must be debug/release/test")
	}
	if cfg.Gin.ReadTimeout <= 0 {
		return fmt.Errorf("gin.read_timeout must be positive")
	}

	// JWT 配置校验
	if cfg.JWT.SecretKey == "" {
		return fmt.Errorf("jwt.secret_key is required")
	}
	if len(cfg.JWT.SecretKey) < 16 {
		return fmt.Errorf("jwt.secret_key must be at least 16 characters")
	}
	switch cfg.JWT.SigningMethod {
	case "HS256", "HS384", "HS512":
	default:
		return fmt.Errorf("jwt.signing_method only support HS256/HS384/HS512")
	}
	if cfg.JWT.ExpireHours <= 0 {
		return fmt.Errorf("jwt.expire_hours must be positive")
	}

	// Email 配置校验
	if cfg.Email.Username != "" || cfg.Email.Password != "" {
		if cfg.Email.SMTPServer == "" {
			return fmt.Errorf("email.smtp_server is required when username/password is set")
		}
		if cfg.Email.From == "" {
			return fmt.Errorf("email.from is required when username/password is set")
		}
	}

	// Login 配置校验
	hasAdmin := cfg.Login.AdminUser != "" && cfg.Login.AdminPassword != ""
	hasUsers := len(cfg.Login.Users) > 0
	if !hasAdmin && !hasUsers {
		return fmt.Errorf("login config: either admin_user/admin_password or users list is required")
	}
	for i, user := range cfg.Login.Users {
		if user.Username == "" {
			return fmt.Errorf("login.users[%d].username is required", i)
		}
		if user.Password == "" {
			return fmt.Errorf("login.users[%d].password is required", i)
		}
		if user.Role == "" {
			return fmt.Errorf("login.users[%d].role is required", i)
		}
	}
	if cfg.App.Env == "prod" {
		if hasAdmin && len(cfg.Login.AdminPassword) < 8 {
			return fmt.Errorf("prod env: login.admin_password must be at least 8 characters")
		}
		for i, user := range cfg.Login.Users {
			if len(user.Password) < 8 {
				return fmt.Errorf("prod env: login.users[%d].password must be at least 8 characters", i)
			}
		}
	}

	return nil
}

// startConfigWatch 启动配置热加载（监听配置文件变化）
func startConfigWatch() {
	// 监听配置文件变化
	cfgViper.WatchConfig()

	// 配置变化回调函数
	cfgViper.OnConfigChange(func(e fsnotify.Event) {
		fmt.Printf("\nconfig file changed: %s (event: %s)\n", e.Name, e.Op)

		// 重新解析并校验配置
		if err := unmarshalAndValidate(); err != nil {
			fmt.Printf("hot-reload config failed: %v\n", err)
			return
		}
	})
}

// ValidateLogin 校验用户名密码（不变）
func (cfg *GlobalConfig) ValidateLogin(username, password string) (bool, string) {
	mu.RLock()
	defer mu.RUnlock()

	if len(cfg.Login.Users) > 0 {
		for _, user := range cfg.Login.Users {
			if user.Username == username && user.Password == password {
				return true, user.Role
			}
		}
		return false, ""
	}

	if cfg.Login.AdminUser == username && cfg.Login.AdminPassword == password {
		return true, "admin"
	}

	return false, ""
}

// SaveDefaultConfig 生成默认配置文件（仅当文件不存在时）
// appName: 传入实际应用名，确保生成的配置与应用名一致
func SaveDefaultConfig(appName string) error {
	configPath := "."
	configFile := filepath.Join(configPath, "config.yaml")

	// 检查文件是否已存在（避免 SafeWriteConfig 的错误）
	if _, err := os.Stat(configFile); err == nil {
		fmt.Printf("default config file already exists: %s, skip generating\n", configFile)
		return nil
	}

	// 复用 cfgViper 的默认值，无需重新创建实例
	cfgViper.Set("app.name", appName)   // 覆盖应用名（与实际检测到的一致）
	cfgViper.Set("jwt.issuer", appName) // 同步 JWT issuer 与应用名

	// 确保配置目录存在（如果路径是 ./configs，自动创建目录）
	if err := os.MkdirAll(configPath, 0755); err != nil {
		return fmt.Errorf("create config dir failed: %v", err)
	}

	// 写入默认配置（SafeWriteConfig 已保证不覆盖）
	if err := cfgViper.SafeWriteConfigAs(configFile); err != nil {
		return fmt.Errorf("write default config failed: %v", err)
	}

	// fmt.Printf("default config file generated successfully: %s\n", configFile)
	return nil
}
