package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"gorm.io/gorm"
)

// 任务类型（int32枚举，保留原有定义）
type TaskType int32

const (
	TaskType_IMMEDIATE TaskType = 0 // 立即执行
	TaskType_CRON      TaskType = 1 // 定时任务（cron表达式）
	TaskType_RECURRING TaskType = 2 // 循环任务（固定间隔）
	TaskType_PERIODIC  TaskType = 3 // 周期定时循环（cron+起止时间）
)

// Enum value maps for TaskType.（保留原有映射表）
var (
	TaskType_name = map[int32]string{
		0: "IMMEDIATE",
		1: "CRON",
		2: "RECURRING",
		3: "PERIODIC",
	}
	TaskType_value = map[string]int32{
		"IMMEDIATE": 0,
		"CRON":      1,
		"RECURRING": 2,
		"PERIODIC":  3,
	}
)

// 实现 Scanner 和 Valuer 接口，适配GORM+SQLite
func (t *TaskType) Scan(value interface{}) error {
	val, ok := value.(int64) // SQLite整数存储为int64
	if !ok {
		return errors.New("invalid TaskType value")
	}
	*t = TaskType(val)
	return nil
}

func (t TaskType) Value() (driver.Value, error) {
	return int64(t), nil // 修复：存储为int64（SQLite兼容）
}

// 任务状态（int32枚举，保留原有定义）
type TaskStatus int32

const (
	TaskStatus_SCHEDULED TaskStatus = 0 // 已调度
	TaskStatus_PAUSED    TaskStatus = 1 // 已暂停
	TaskStatus_STOPPED   TaskStatus = 2 // 已停止
)

// Enum value maps for TaskStatus.（保留原有映射表）
var (
	TaskStatus_name = map[int32]string{
		0: "SCHEDULED",
		1: "PAUSED",
		2: "STOPPED",
	}
	TaskStatus_value = map[string]int32{
		"SCHEDULED": 0,
		"PAUSED":    1,
		"STOPPED":   2,
	}
)

// 实现 Scanner 和 Valuer 接口，适配GORM+SQLite
func (s *TaskStatus) Scan(value interface{}) error {
	val, ok := value.(int64)
	if !ok {
		return errors.New("invalid TaskStatus value")
	}
	*s = TaskStatus(val)
	return nil
}

func (s TaskStatus) Value() (driver.Value, error) {
	return int64(s), nil // 修复：存储为int64（SQLite兼容）
}

// 探测类型（int32枚举，保留原有定义）
type ProbeType int32

const (
	ProbeType_HTTP ProbeType = 0
	ProbeType_TCP  ProbeType = 1
	ProbeType_DNS  ProbeType = 2
	ProbeType_PING ProbeType = 3
)

// Enum value maps for ProbeType.（保留原有映射表）
var (
	ProbeType_name = map[int32]string{
		0: "HTTP",
		1: "TCP",
		2: "DNS",
		3: "PING",
	}
	ProbeType_value = map[string]int32{
		"HTTP": 0,
		"TCP":  1,
		"DNS":  2,
		"PING": 3,
	}
)

// 实现 Scanner 和 Valuer 接口，适配GORM+SQLite
func (p *ProbeType) Scan(value interface{}) error {
	val, ok := value.(int64)
	if !ok {
		return errors.New("invalid ProbeType value")
	}
	*p = ProbeType(val)
	return nil
}

func (p ProbeType) Value() (driver.Value, error) {
	return int64(p), nil // 修复：存储为int64（SQLite兼容）
}

// 任务结构体（GORM+SQLite适配版）
type Task struct {
	ID             string         `gorm:"column:id;type:VARCHAR(64);primaryKey;uniqueIndex;not null" json:"id"`      // 任务ID（主键，建议用UUID）
	Name           string         `gorm:"column:name;type:VARCHAR(128);uniqueIndex;not null" json:"name"`            // 任务名称
	Type           TaskType       `gorm:"column:type;type:INTEGER;not null;default:0" json:"type"`                   // 任务类型（int32）
	ProbeConfig    *ProbeConfig   `gorm:"column:probe_config;type:JSON;not null" json:"probe_config"`                // 探测配置（JSON序列化）
	CronExpression string         `gorm:"column:cron_expression;type:VARCHAR(64);default:''" json:"cron_expression"` // Cron表达式（仅Cron/PERIODIC类型用）
	StartTime      int64          `gorm:"column:start_time;type:INTEGER;default:0" json:"start_time"`                // 开始时间戳（仅PERIODIC类型用）
	EndTime        int64          `gorm:"column:end_time;type:INTEGER;default:0" json:"end_time"`                    // 结束时间戳（仅PERIODIC类型用）
	Status         TaskStatus     `gorm:"column:status;type:INTEGER;not null;default:0" json:"status"`               // 任务状态（默认已调度）
	CreatedAt      time.Time      `gorm:"column:created_at;autoCreateTime" json:"created_at"`                        // 创建时间（自动填充）
	UpdatedAt      time.Time      `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`                        // 更新时间（自动填充）
	DeletedAt      gorm.DeletedAt `gorm:"column:deleted_at;index" json:"-"`                                          // 软删除标记
}

// 探测配置结构体（GORM+SQLite适配版）
type ProbeConfig struct {
	Type                  ProbeType `json:"type"`                    // 探测类型（int32枚举）
	Target                string    `json:"target"`                  // 探测目标（IP/域名）
	Tls                   bool      `json:"tls"`                     // 是否启用TLS
	Port                  int64     `json:"port"`                    // 探测端口
	Timeout               int64     `json:"timeout"`                 // 超时时间（毫秒）
	FailureWindowDuration int64     `json:"failure_window_duration"` // 失败窗口时长（秒）
	SuccessWindowDuration int64     `json:"success_window_duration"` // 成功窗口时长（秒）
	FailureThreshold      int       `json:"failure_threshold"`       // 失败阈值（连续失败次数）
	AlertEmailRecipients  []string  `json:"alert_email_recipients"`  // 告警邮件接收人列表
}

// 实现 ProbeConfig 的 JSON 序列化接口，适配SQLite的JSON/TEXT类型
func (pc *ProbeConfig) Scan(value interface{}) error {
	val, ok := value.([]byte)
	if !ok {
		return errors.New("invalid ProbeConfig value")
	}
	return json.Unmarshal(val, pc)
}

func (pc ProbeConfig) Value() (driver.Value, error) {
	return json.Marshal(pc)
}

// TableName 指定任务表名（SQLite表名默认小写，可自定义）
func (Task) TableName() string {
	return "tasks"
}
