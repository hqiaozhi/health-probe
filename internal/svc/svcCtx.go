package svc

import (
	"context"
	"fmt"
	"health-probe/internal/conf"
	"health-probe/internal/core/metrics"
	"health-probe/internal/core/notify/email"
	"health-probe/internal/core/rule"
	"health-probe/internal/core/scheduler"
	"health-probe/internal/core/tools"
	"health-probe/internal/core/tools/dns"
	"health-probe/internal/core/tools/http"
	"health-probe/internal/core/tools/ping"
	"health-probe/internal/core/tools/tcp"
	"health-probe/internal/logger"
	"health-probe/internal/models"
	"health-probe/internal/utils/jwt"
	"health-probe/internal/utils/resp"
	"os"

	"log"

	gormLogger "gorm.io/gorm/logger"

	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

type ServiceContext struct {
	Tools     tools.Tools
	Conf      conf.GlobalConfig
	DB        *gorm.DB
	Metrics   *metrics.Metrics
	Email     email.Sender
	Rule      *rule.MemoryAlertStatService
	Executor  *scheduler.Executor
	Scheduler scheduler.IScheduler
	Logger    *zap.Logger
	DAO       *models.DAO
	JWT       *jwt.JwtService
	Resp      *resp.Resp
}

func NewSvcCtx(ctx context.Context) *ServiceContext {
	// 初始化ServiceContext
	s := &ServiceContext{}

	// 加载配置
	conf := conf.Loaded()
	s.Conf = conf

	// 初始化JWT服务
	s.JWT = jwt.NewJWTService(&s.Conf.JWT)

	// 初始化数据库
	sqliteCfg := conf.DB.SQLite
	gormCfg := conf.DB.GORM
	dsn := fmt.Sprintf(
		"file:%s?cache=shared&_journal_mode=%s&_cache_size=%d&_synchronous=%s",
		sqliteCfg.Path,
		getDefaultVal(sqliteCfg.JournalMode, "WAL"),  // 配置为空时用默认值
		getDefaultVal(sqliteCfg.CacheSize, 2048),     // 缓存默认2048KB
		getDefaultVal(sqliteCfg.Synchronous, "FULL"), // 同步模式默认FULL
	)
	gormOptions := &gorm.Config{
		Logger: gormLogger.Default.LogMode(parseLogMode(gormCfg.LogMode)),
		NamingStrategy: schema.NamingStrategy{
			SingularTable: true, // 表名不加s（如 tasks → task）
		},
		// 禁用外键（从config.SQLite.DisableForeignKey读取）
		DisableForeignKeyConstraintWhenMigrating: sqliteCfg.DisableForeignKey,
	}
	db, err := gorm.Open(sqlite.Open(dsn), gormOptions)
	if err != nil {
		panic(fmt.Errorf("connect sqlite failed: %w", err))
	}
	sqlDB, err := db.DB()
	if err != nil {
		panic(fmt.Errorf("get sql.DB instance failed: %w", err))
	}
	sqlDB.SetMaxOpenConns(gormCfg.MaxOpenConns)       // 最大打开连接数
	sqlDB.SetMaxIdleConns(gormCfg.MaxIdleConns)       // 最大空闲连接数
	sqlDB.SetConnMaxLifetime(gormCfg.ConnMaxLifetime) // 连接最大存活时间
	sqlDB.SetConnMaxIdleTime(gormCfg.ConnMaxIdleTime) // 连接最大空闲时间

	if err := sqlDB.PingContext(ctx); err != nil {
		panic(fmt.Errorf("ping sqlite failed: %w", err))
	}

	// 自动迁移表结构（从config.SQLite.AutoMigrate读取）
	if sqliteCfg.AutoMigrate {
		if err := db.WithContext(ctx).AutoMigrate(&models.Task{}, &models.EmailLog{}); err != nil {
			panic(fmt.Errorf("auto migrate table failed: %w", err))
		}
	}
	s.DB = db

	// 初始化DAO
	DAO := models.NewDAO(db)
	s.DAO = DAO

	// 初始化日志
	if err := logger.InitLogger(conf.Log); err != nil {
		fmt.Printf("init logger failed: %v", err)
		os.Exit(1)
	}
	s.Logger = logger.Logger

	// 初始化指标收集器
	metrics := metrics.New()
	s.Metrics = metrics

	// 初始化邮件发送器
	email, err := email.NewSender(&conf.Email, s.Logger)
	if err != nil {
		s.Logger.Error("create email sender failed: " + err.Error())
		os.Exit(1)
	}
	s.Email = email

	// 初始化告警统计服务
	ruleService, err := rule.NewMemoryAlertStatService(DAO, email, 10000, s.Logger)
	if err != nil {
		s.Logger.Error("create rule service failed: " + err.Error())
		os.Exit(1)
	}
	s.Rule = ruleService

	// 初始化工具类
	tools := tools.Tools{
		HTTP: http.New(),
		TCP:  tcp.New(),
		DNS:  dns.New(),
		PING: ping.New(),
	}

	// 初始化执行器
	executor := scheduler.NewExecutor(tools, ruleService, metrics, s.Logger)
	s.Executor = executor

	// 初始化调度器
	scheduler := scheduler.NewScheduler(executor)
	s.Scheduler = scheduler

	// 初始化响应工具
	s.Resp = resp.New()

	// 加载任务
	s.Scheduler.Start()
	s.LoadTasks()

	return s
}

// 辅助函数：获取默认值（配置为空时使用）
func getDefaultVal[T comparable](val, def T) T {
	var zero T
	if val == zero {
		return def
	}
	return val
}

// 辅助函数：解析GORM日志级别
func parseLogMode(mode string) gormLogger.LogLevel {
	switch mode {
	case "silent":
		return gormLogger.Silent
	case "error":
		return gormLogger.Error
	case "warn":
		return gormLogger.Warn
	case "info":
		return gormLogger.Info
	default:
		return gormLogger.Info // 默认info级别
	}
}

// 从数据库中加载探测任务
func (s *ServiceContext) LoadTasks() {
	var tasks []*models.Task
	status := models.TaskStatus_SCHEDULED
	tasks, _, err := s.DAO.Task.ListByCondition(context.Background(), models.TaskQueryCondition{
		Status:    &status,
		Page:      0,
		PageSize:  99999,
		SortField: "id",
		SortOrder: "asc",
	})
	if err != nil {
		s.Logger.Error("find all tasks failed: " + err.Error())
	}
	// 加载计数
	count := 0
	for _, task := range tasks {
		taskID, err := s.Scheduler.CreateTask(context.Background(), &scheduler.Task{
			ID:   task.ID,
			Name: task.Name,
			Type: scheduler.TaskType(task.Type),
			ProbeConfig: &scheduler.ProbeConfig{
				Type:                  scheduler.ProbeType(task.ProbeConfig.Type),
				Target:                task.ProbeConfig.Target,
				Port:                  task.ProbeConfig.Port,
				Tls:                   task.ProbeConfig.Tls,
				Timeout:               task.ProbeConfig.Timeout,
				FailureWindowDuration: task.ProbeConfig.FailureWindowDuration,
				SuccessWindowDuration: task.ProbeConfig.SuccessWindowDuration,
				FailureThreshold:      task.ProbeConfig.FailureThreshold,
				AlertEmailRecipients:  task.ProbeConfig.AlertEmailRecipients,
			},
			CronExpression: task.CronExpression,
			StartTime:      task.StartTime,
			EndTime:        task.EndTime,
			Status:         scheduler.TaskStatus(task.Status),
		})
		count++
		if err != nil {
			s.Logger.Error("create task " + taskID + " failed: " + err.Error())
		}
	}
	// 记录加载任务数
	log.Printf("Loading tasks count: %d", count)
}
