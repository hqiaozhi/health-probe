package models

import (
	"context"
	"errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// EmailLogDAO 邮件日志DAO接口
type EmailLogDAO interface {
	// 单条操作
	Create(ctx context.Context, log *EmailLog) error
	GetByID(ctx context.Context, logID string) (*EmailLog, error)
	Update(ctx context.Context, log *EmailLog) error
	DeleteByID(ctx context.Context, logID string) error

	// 批量操作
	BatchCreate(ctx context.Context, logs []*EmailLog) error
	BatchGetByIDs(ctx context.Context, logIDs []string) ([]*EmailLog, error)
	BatchDeleteByIDs(ctx context.Context, logIDs []string) error

	// 条件查询（支持多条件+分页）
	ListByCondition(ctx context.Context, condition EmailLogQueryCondition) ([]*EmailLog, int64, error)

	// 更新已读状态
	UpdateReadStatus(ctx context.Context, ID string) error
}

// EmailLogQueryCondition 条件查询参数
type EmailLogQueryCondition struct {
	TaskID         *string         `json:"task_id"`          // 任务ID（可选）
	TaskName       *string         `json:"task_name"`        // 任务名称（新增，可选）
	ProbeType      *ProbeType      `json:"probe_type"`       // 探测类型（新增，可选）
	Target         *string         `json:"target"`           // 探测目标（可选，模糊查询）
	AlertType      *AlertType      `json:"alert_type"`       // 告警类型（可选）
	SendStatus     *EmailLogStatus `json:"send_status"`      // 发送状态（可选）
	Is_Read        *bool           `json:"is_read"`          // 是否已读（可选）
	Page           int             `json:"page"`             // 页码（默认1）
	PageSize       int             `json:"page_size"`        // 每页条数（默认20，最大100）
	WithSoftDelete bool            `json:"with_soft_delete"` // 是否包含软删除（默认不包含）
}

// emailLogDAO 实现 EmailLogDAO 接口
type emailLogDAO struct {
	db *gorm.DB
}

// NewEmailLogDAO 创建 EmailLogDAO 实例
func NewEmailLogDAO(db *gorm.DB) EmailLogDAO {
	return &emailLogDAO{db: db}
}

// -------------------------- 单条操作 --------------------------

// Create 新增单条邮件日志
func (d *emailLogDAO) Create(ctx context.Context, log *EmailLog) error {
	if log == nil || log.ID == "" {
		return errors.New("email log is nil or logID is empty")
	}
	return d.db.WithContext(ctx).Create(log).Error
}

// GetByID 根据ID查询单条日志
func (d *emailLogDAO) GetByID(ctx context.Context, logID string) (*EmailLog, error) {
	if logID == "" {
		return nil, errors.New("logID is empty")
	}
	var log EmailLog
	tx := d.db.WithContext(ctx).Where("id = ?", logID)
	if err := tx.First(&log).Error; err != nil {
		return nil, err
	}
	return &log, nil
}

// Update 全量更新单条日志（需传入完整对象）
func (d *emailLogDAO) Update(ctx context.Context, log *EmailLog) error {
	if log == nil || log.ID == "" {
		return errors.New("email log is nil or logID is empty")
	}
	return d.db.WithContext(ctx).Model(log).Select(clause.Associations).Updates(log).Error
}

// DeleteByID 物理删除单条日志
func (d *emailLogDAO) DeleteByID(ctx context.Context, logID string) error {
	if logID == "" {
		return errors.New("logID is empty")
	}
	return d.db.WithContext(ctx).Unscoped().Delete(&EmailLog{}, "id = ?", logID).Error
}

// -------------------------- 批量操作 --------------------------

// BatchCreate 批量新增邮件日志
func (d *emailLogDAO) BatchCreate(ctx context.Context, logs []*EmailLog) error {
	if len(logs) == 0 {
		return errors.New("email logs is empty")
	}
	// 校验ID非空
	for _, log := range logs {
		if log.ID == "" {
			return errors.New("logID is empty in batch create")
		}
	}
	// 分批次插入（每100条一批）
	return d.db.WithContext(ctx).CreateInBatches(logs, 100).Error
}

// BatchGetByIDs 批量根据ID查询日志
func (d *emailLogDAO) BatchGetByIDs(ctx context.Context, logIDs []string) ([]*EmailLog, error) {
	if len(logIDs) == 0 {
		return nil, errors.New("logIDs is empty")
	}
	var logs []*EmailLog
	if err := d.db.WithContext(ctx).
		Where("id IN (?)", logIDs).
		Find(&logs).Error; err != nil {
		return nil, err
	}
	return logs, nil
}

// BatchDeleteByIDs 批量软删除日志
func (d *emailLogDAO) BatchDeleteByIDs(ctx context.Context, logIDs []string) error {
	if len(logIDs) == 0 {
		return errors.New("logIDs is empty")
	}
	// 判断ID是否存在
	logs, err := d.BatchGetByIDs(ctx, logIDs)
	if err != nil {
		return err
	}
	if len(logs) != len(logIDs) {
		return errors.New("some logIDs do not exist")
	}

	return d.db.WithContext(ctx).Unscoped().Delete(&EmailLog{}, "id IN (?)", logIDs).Error
}

// -------------------------- 条件查询 --------------------------

// ListByCondition 条件查询邮件日志（支持分页）
func (d *emailLogDAO) ListByCondition(ctx context.Context, condition EmailLogQueryCondition) ([]*EmailLog, int64, error) {
	tx := d.db.WithContext(ctx).Model(&EmailLog{})

	// 排除软删除数据（默认）
	if !condition.WithSoftDelete {
		tx = tx.Where("deleted_at IS NULL")
	}

	// 条件筛选：任务ID
	if condition.TaskID != nil && *condition.TaskID != "" {
		tx = tx.Where("task_id = ?", *condition.TaskID)
	}

	// 条件筛选：任务名称（新增）
	if condition.TaskName != nil && *condition.TaskName != "" {
		tx = tx.Where("task_name LIKE ?", "%"+*condition.TaskName+"%")
	}

	// 条件筛选：探测类型（新增）
	if condition.ProbeType != nil {
		tx = tx.Where("probe_type = ?", *condition.ProbeType)
	}

	// 条件筛选：探测目标（模糊查询）
	if condition.Target != nil && *condition.Target != "" {
		tx = tx.Where("target LIKE ?", "%"+*condition.Target+"%")
	}

	// 条件筛选：告警类型
	if condition.AlertType != nil {
		tx = tx.Where("alert_type = ?", *condition.AlertType)
	}

	// 条件筛选：发送状态
	if condition.SendStatus != nil {
		tx = tx.Where("send_status = ?", *condition.SendStatus)
	}

	// 条件筛选：是否已读
	if condition.Is_Read != nil {
		tx = tx.Where("is_read = ?", *condition.Is_Read)
	}

	// 统计总条数
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页处理
	page := condition.Page
	pageSize := condition.PageSize
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize
	tx = tx.Offset(offset).Limit(pageSize)

	// 排序：默认按发送时间倒序（最新的在前）
	tx = tx.Order("created_at DESC")

	// 执行查询
	var logs []*EmailLog
	if err := tx.Find(&logs).Error; err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}

// 更新已读状态
func (d *emailLogDAO) UpdateReadStatus(ctx context.Context, ID string) error {
	if ID == "" {
		return errors.New("ID is empty")
	}

	// 只更新需要的字段
	updates := map[string]interface{}{
		"is_read": true,
	}

	return d.db.WithContext(ctx).Model(&EmailLog{}).Where("id = ?", ID).Updates(updates).Error
}
