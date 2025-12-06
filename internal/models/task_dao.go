package models

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
)

// TaskDAO 任务DAO层接口（定义所有操作方法）
type TaskDAO interface {
	// 单条操作
	Create(ctx context.Context, task *Task) error
	GetByID(ctx context.Context, taskID string) (*Task, error)
	Update(ctx context.Context, task *Task) error
	UpdateStatus(ctx context.Context, taskID string, status TaskStatus) error
	DeleteByID(ctx context.Context, taskID string) error

	// 批量操作
	BatchCreate(ctx context.Context, tasks []*Task) error
	BatchGetByIDs(ctx context.Context, taskIDs []string) ([]*Task, error)
	BatchUpdate(ctx context.Context, tasks []*Task) error
	BatchUpdateStatus(ctx context.Context, taskIDs []string, status TaskStatus) error
	BatchDeleteByIDs(ctx context.Context, taskIDs []string) error

	// 条件查询
	ListByCondition(ctx context.Context, condition TaskQueryCondition) ([]*Task, int64, error)

	// 新增：简单分页查询
	ListWithPagination(ctx context.Context, page, pageSize int, sortField, sortOrder string) ([]*Task, int64, error)
}

// TaskQueryCondition 条件查询参数
type TaskQueryCondition struct {
	IDs            []string    // 任务ID列表（批量查询）
	Name           *string     // 任务名称（可选）
	TaskType       *TaskType   // 任务类型（可选）
	Status         *TaskStatus // 任务状态（可选）
	ProbeType      *ProbeType  // 探测类型（可选，需通过JSON筛选）
	Target         *string     // 探测目标（可选，需通过JSON筛选）
	StartTimeMin   int64       // 开始时间最小值（可选）
	StartTimeMax   int64       // 开始时间最大值（可选）
	Page           int         // 页码（默认1）
	PageSize       int         // 每页条数（默认20，最大100）
	SortField      string      // 排序字段（可选，默认created_at）
	SortOrder      string      // 排序方向（可选，默认DESC）
	WithSoftDelete bool        // 是否包含软删除数据（默认不包含）
}

// taskDAO 实现TaskDAO接口
type taskDAO struct {
	db *gorm.DB // GORM数据库连接
}

// NewTaskDAO 创建TaskDAO实例
func NewTaskDAO(db *gorm.DB) TaskDAO {
	return &taskDAO{db: db}
}

// -------------------------- 单条操作 --------------------------

// Create 新增单条任务
func (d *taskDAO) Create(ctx context.Context, task *Task) error {
	if task == nil || task.ID == "" {
		return errors.New("task is nil or taskID is empty")
	}
	return d.db.WithContext(ctx).Create(task).Error
}

// GetByID 根据ID查询单条任务
func (d *taskDAO) GetByID(ctx context.Context, taskID string) (*Task, error) {
	if taskID == "" {
		return nil, errors.New("taskID is empty")
	}
	var task Task
	tx := d.db.WithContext(ctx).Where("id = ?", taskID)
	if err := tx.First(&task).Error; err != nil {
		return nil, err
	}
	return &task, nil
}

// Update 部分更新单条任务（精确控制更新字段）
func (d *taskDAO) Update(ctx context.Context, task *Task) error {
	if task == nil || task.ID == "" {
		return errors.New("task is nil or taskID is empty")
	}

	// 构建更新map，只包含需要更新的字段
	updateData := make(map[string]interface{})

	// 只更新有值的字段（避免更新零值）
	if task.Name != "" {
		updateData["name"] = task.Name
	}
	if task.Type != 0 {
		updateData["type"] = task.Type
	}
	if task.ProbeConfig != nil {
		updateData["probe_config"] = task.ProbeConfig
	}
	if task.CronExpression != "" {
		updateData["cron_expression"] = task.CronExpression
	}

	// 总是更新updated_at字段
	updateData["updated_at"] = time.Now()

	// 如果没有任何字段需要更新，返回错误
	if len(updateData) == 1 { // 只有updated_at
		return errors.New("no fields to update")
	}

	// 执行更新
	return d.db.WithContext(ctx).Model(&Task{}).Where("id = ?", task.ID).Updates(updateData).Error
}

// UpdateStatus 单独更新任务状态（常用场景优化）
func (d *taskDAO) UpdateStatus(ctx context.Context, taskID string, status TaskStatus) error {
	if taskID == "" {
		return errors.New("taskID is empty")
	}
	return d.db.WithContext(ctx).Model(&Task{}).
		Where("id = ?", taskID).
		Update("status", status).Error
}

// DeleteByID 软删除单条任务
func (d *taskDAO) DeleteByID(ctx context.Context, taskID string) error {
	if taskID == "" {
		return errors.New("taskID is empty")
	}
	// 使用Unscoped()硬删除：直接从数据库中删除记录
	return d.db.WithContext(ctx).Unscoped().Delete(&Task{}, "id = ?", taskID).Error
}

// -------------------------- 批量操作 --------------------------

// BatchCreate 批量新增任务（支持批量插入优化）
func (d *taskDAO) BatchCreate(ctx context.Context, tasks []*Task) error {
	if len(tasks) == 0 {
		return errors.New("tasks is empty")
	}

	// SQLite支持批量插入，GORM自动优化
	return d.db.WithContext(ctx).CreateInBatches(tasks, 100).Error // 每100条分一批（可调整）
}

// BatchGetByIDs 批量根据ID查询任务
func (d *taskDAO) BatchGetByIDs(ctx context.Context, taskIDs []string) ([]*Task, error) {
	if len(taskIDs) == 0 {
		return nil, errors.New("taskIDs is empty")
	}
	var tasks []*Task
	if err := d.db.WithContext(ctx).
		Where("id IN (?)", taskIDs).
		Find(&tasks).Error; err != nil {
		return nil, err
	}
	return tasks, nil
}

// BatchUpdate 批量部分更新任务（只更新传入的字段）
func (d *taskDAO) BatchUpdate(ctx context.Context, tasks []*Task) error {
	if len(tasks) == 0 {
		return errors.New("tasks is empty")
	}
	// 批量更新：使用事务保证原子性
	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, task := range tasks {
			if task.ID == "" {
				return errors.New("taskID is empty in batch update")
			}

			// 构建更新map，只包含非空字段
			updateData := make(map[string]interface{})

			// 只更新有值的字段
			if task.Name != "" {
				updateData["name"] = task.Name
			}
			if task.Type != 0 {
				updateData["type"] = task.Type
			}
			if task.ProbeConfig != nil {
				updateData["probe_config"] = task.ProbeConfig
			}
			if task.CronExpression != "" {
				updateData["cron_expression"] = task.CronExpression
			}
			if task.StartTime != 0 {
				updateData["start_time"] = task.StartTime
			}
			if task.EndTime != 0 {
				updateData["end_time"] = task.EndTime
			}
			if task.Status != 0 {
				updateData["status"] = task.Status
			}

			// 总是更新updated_at字段
			updateData["updated_at"] = time.Now()

			// 如果没有任何字段需要更新，跳过
			if len(updateData) == 1 { // 只有updated_at
				continue
			}

			// 执行更新
			if err := tx.Model(&Task{}).Where("id = ?", task.ID).Updates(updateData).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// BatchUpdateStatus 批量更新任务状态（常用场景优化）
func (d *taskDAO) BatchUpdateStatus(ctx context.Context, taskIDs []string, status TaskStatus) error {
	if len(taskIDs) == 0 {
		return errors.New("taskIDs is empty")
	}
	return d.db.WithContext(ctx).Model(&Task{}).
		Where("id IN (?)", taskIDs).
		Update("status", status).Error
}

// BatchDeleteByIDs 批量软删除任务
func (d *taskDAO) BatchDeleteByIDs(ctx context.Context, taskIDs []string) error {
	if len(taskIDs) == 0 {
		return errors.New("taskIDs is empty")
	}
	return d.db.WithContext(ctx).Unscoped().Delete(&Task{}, "id IN (?)", taskIDs).Error
}

// -------------------------- 条件查询 --------------------------

// ListByCondition 条件查询任务（支持分页、多条件筛选）
func (d *taskDAO) ListByCondition(ctx context.Context, condition TaskQueryCondition) ([]*Task, int64, error) {
	tx := d.db.WithContext(ctx).Model(&Task{})

	// 处理软删除：默认不包含，需要则显示指定Unscoped()
	if !condition.WithSoftDelete {
		tx = tx.Where("deleted_at IS NULL")
	}

	// 任务名称（模糊查询）
	if condition.Name != nil && *condition.Name != "" {
		tx = tx.Where("name LIKE ?", "%"+*condition.Name+"%")
	}

	// 条件筛选：任务类型
	if condition.TaskType != nil {
		tx = tx.Where("type = ?", *condition.TaskType)
	}

	// 条件筛选：任务状态
	if condition.Status != nil {
		tx = tx.Where("status = ?", *condition.Status)
	}

	// 条件筛选：探测类型（SQLite JSON字段筛选，需用json_extract）
	if condition.ProbeType != nil {
		// probe_config -> '$.type' 等价于 json_extract(probe_config, '$.type')
		tx = tx.Where("json_extract(probe_config, '$.type') = ?", *condition.ProbeType)
	}

	// 条件筛选：探测目标（模糊查询）
	if condition.Target != nil && *condition.Target != "" {
		tx = tx.Where("json_extract(probe_config, '$.target') LIKE ?", "%"+*condition.Target+"%")
	}

	// 条件筛选：开始时间范围
	if condition.StartTimeMin > 0 {
		tx = tx.Where("start_time >= ?", condition.StartTimeMin)
	}
	if condition.StartTimeMax > 0 {
		tx = tx.Where("start_time <= ?", condition.StartTimeMax)
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
		pageSize = 20 // 限制最大每页100条
	}
	offset := (page - 1) * pageSize
	tx = tx.Offset(offset).Limit(pageSize)

	// 排序处理
	sortField := condition.SortField
	sortOrder := condition.SortOrder
	if sortField == "" {
		sortField = "created_at" // 默认排序字段
	}
	if sortOrder == "" {
		sortOrder = "DESC" // 默认排序方向
	}

	// 验证排序字段和方向
	validSortFields := map[string]bool{
		"created_at": true,
		"updated_at": true,
		"start_time": true,
		"status":     true,
		"type":       true,
	}

	validSortOrders := map[string]bool{
		"ASC":  true,
		"DESC": true,
	}

	if !validSortFields[sortField] {
		sortField = "created_at" // 无效字段使用默认值
	}
	if !validSortOrders[sortOrder] {
		sortOrder = "DESC" // 无效方向使用默认值
	}

	tx = tx.Order(sortField + " " + sortOrder)

	// 执行查询
	var tasks []*Task
	if err := tx.Find(&tasks).Error; err != nil {
		return nil, 0, err
	}

	return tasks, total, nil
}

// -------------------------- 简单分页查询 --------------------------

// ListWithPagination 简单分页查询任务列表（支持排序）
func (d *taskDAO) ListWithPagination(ctx context.Context, page, pageSize int, sortField, sortOrder string) ([]*Task, int64, error) {
	// 参数校验和默认值设置
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20 // 默认每页20条
	}
	if pageSize > 100 {
		pageSize = 100 // 限制最大每页100条
	}

	// 计算偏移量
	offset := (page - 1) * pageSize

	// 统计总条数
	var total int64
	tx := d.db.WithContext(ctx).Model(&Task{}).Where("deleted_at IS NULL")
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 排序处理
	if sortField == "" {
		sortField = "created_at" // 默认排序字段
	}
	if sortOrder == "" {
		sortOrder = "DESC" // 默认排序方向
	}

	// 验证排序字段和方向
	validSortFields := map[string]bool{
		"created_at": true,
		"updated_at": true,
		"start_time": true,
		"status":     true,
		"type":       true,
	}

	validSortOrders := map[string]bool{
		"ASC":  true,
		"DESC": true,
	}

	if !validSortFields[sortField] {
		sortField = "created_at" // 无效字段使用默认值
	}
	if !validSortOrders[sortOrder] {
		sortOrder = "DESC" // 无效方向使用默认值
	}

	// 执行分页查询
	var tasks []*Task
	tx = d.db.WithContext(ctx).Model(&Task{}).
		Where("deleted_at IS NULL").
		Order(sortField + " " + sortOrder).
		Offset(offset).
		Limit(pageSize)

	if err := tx.Find(&tasks).Error; err != nil {
		return nil, 0, err
	}

	return tasks, total, nil
}
