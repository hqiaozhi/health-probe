package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"gorm.io/gorm"
)

// EmailLogStatus 邮件发送状态（成功/失败）
type EmailLogStatus int32

const (
	EmailLogStatus_Success EmailLogStatus = 0 // 发送成功
	EmailLogStatus_Failed  EmailLogStatus = 1 // 发送失败
)

// EmailLogStatus 枚举映射（序列化/反序列化用）
var (
	EmailLogStatus_name = map[int32]string{
		0: "SUCCESS",
		1: "FAILED",
	}
	EmailLogStatus_value = map[string]int32{
		"SUCCESS": 0,
		"FAILED":  1,
	}
)

// 实现 Scanner 和 Valuer 接口，适配 GORM + SQLite
func (s *EmailLogStatus) Scan(value interface{}) error {
	val, ok := value.(int64)
	if !ok {
		return errors.New("invalid EmailLogStatus value")
	}
	*s = EmailLogStatus(val)
	return nil
}

func (s EmailLogStatus) Value() (driver.Value, error) {
	return int64(s), nil // 修改为返回 int64
}

// AlertType 告警类型（失败/恢复）
type AlertType int32

const (
	AlertType_Failure  AlertType = 0 // 探测失败告警
	AlertType_Recovery AlertType = 1 // 探测恢复告警
)

// AlertType 枚举映射
var (
	AlertType_name = map[int32]string{
		0: "FAILURE",
		1: "RECOVERY",
	}
	AlertType_value = map[string]int32{
		"FAILURE":  0,
		"RECOVERY": 1,
	}
)

// 实现 Scanner 和 Valuer 接口，适配 GORM + SQLite
func (a *AlertType) Scan(value interface{}) error {
	val, ok := value.(int64)
	if !ok {
		return errors.New("invalid AlertType value")
	}
	*a = AlertType(val)
	return nil
}

func (a AlertType) Value() (driver.Value, error) {
	return int64(a), nil // 修改为返回 int64
}

// EmailLog 邮件发送记录表模型
type EmailLog struct {
	ID         string         `gorm:"column:id;type:VARCHAR(64);primaryKey;not null" json:"id"`          // 日志ID（建议UUID）
	TaskID     string         `gorm:"column:task_id;type:VARCHAR(64);not null;index" json:"task_id"`     // 关联的任务ID
	TaskName   string         `gorm:"column:task_name;type:VARCHAR(128);not null" json:"task_name"`      // 任务名称
	ProbeType  ProbeType      `gorm:"column:probe_type;type:INTEGER;not null;index" json:"probe_type"`   // 探测类型（新增字段）
	Target     string         `gorm:"column:target;type:VARCHAR(128);not null;index" json:"target"`      // 探测目标（IP/域名）
	AlertType  AlertType      `gorm:"column:alert_type;type:INTEGER;not null;index" json:"alert_type"`   // 告警类型（失败/恢复）
	SendStatus EmailLogStatus `gorm:"column:send_status;type:INTEGER;not null;index" json:"send_status"` // 发送状态（成功/失败）
	Recipients Recipients     `gorm:"column:recipients;type:JSON;not null" json:"recipients"`            // 收件人列表（JSON序列化）
	Message    string         `gorm:"column:message;type:TEXT;default:''" json:"message"`                // 详细信息
	Is_Read    bool           `gorm:"column:is_read;type:BOOLEAN;default:false" json:"is_read"`          // 是否已读
	CreatedAt  time.Time      `gorm:"column:created_at;autoCreateTime" json:"created_at"`                // 创建时间（自动填充）
	UpdatedAt  time.Time      `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`                // 更新时间（自动填充）
	DeletedAt  gorm.DeletedAt `gorm:"column:deleted_at;index" json:"-"`                                  // 软删除标记
}

// Recipients 自定义类型用于处理收件人列表的序列化
type Recipients []string

// 实现 Scanner 和 Valuer 接口
func (r *Recipients) Scan(value interface{}) error {
	if value == nil {
		*r = Recipients{}
		return nil
	}

	// 处理字符串类型的JSON
	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, r)
	case string:
		return json.Unmarshal([]byte(v), r)
	default:
		return errors.New("invalid recipients value type")
	}
}

func (r Recipients) Value() (driver.Value, error) {
	if len(r) == 0 {
		return "[]", nil
	}
	return json.Marshal(r)
}

// TableName 指定表名
func (EmailLog) TableName() string {
	return "email_logs"
}

// 删除结构体级别的 Scan 和 Value 方法，让 GORM 使用字段级别的序列化
// func (el *EmailLog) Scan(value interface{}) error {
//     return nil
// }
//
// func (el EmailLog) Value() (driver.Value, error) {
//     return json.Marshal(el.Recipients)
// }

// 辅助方法：获取告警类型字符串
func (a AlertType) String() string {
	return AlertType_name[int32(a)]
}

// 辅助方法：获取发送状态字符串
func (s EmailLogStatus) String() string {
	return EmailLogStatus_name[int32(s)]
}
