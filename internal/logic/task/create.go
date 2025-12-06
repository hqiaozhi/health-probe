package task

import (
	"context"
	"fmt"
	"health-probe/internal/core/scheduler"
	"health-probe/internal/models"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// BatchCreate 创建多个任务（先添加到数据库，再添加到调度器）
// 注意：所有任务都是RECURRING类型（固定为循环任务）
func (t *Task) BatchCreate(ctx context.Context, req []*models.Task) error {
	var Tasks []*models.Task
	// 先验证所有任务的参数
	for _, task := range req {
		// 生成任务ID
		task_id, err := uuid.NewV7()
		if err != nil {
			return err
		}
		schedulerTask := &models.Task{
			ID:   task_id.String(),
			Name: task.Name,
			Type: models.TaskType(2),
			ProbeConfig: &models.ProbeConfig{
				Type:                  models.ProbeType(task.ProbeConfig.Type),
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
			Status:         models.TaskStatus(task.Status),
		}

		// 验证调度器参数（但不实际创建）
		if err := t.validateSchedulerTask(schedulerTask); err != nil {
			return fmt.Errorf("task %s validation failed: %v", task.ID, err)
		}
		Tasks = append(Tasks, schedulerTask)
	}

	// 先添加到数据库
	err := t.svcCtx.DAO.Task.BatchCreate(ctx, Tasks)
	if err != nil {
		t.svcCtx.Logger.Error("Database create task failed", zap.Error(err))
		return err
	}

	// 再添加到调度器
	var failedTasks []string
	for _, task := range Tasks {
		schedulerTask := &scheduler.Task{
			ID:   task.ID,
			Name: task.Name,
			Type: scheduler.TaskType(2),
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
			Status:         scheduler.TaskStatus(task.Status),
		}

		taskID, err := t.svcCtx.Scheduler.CreateTask(ctx, schedulerTask)
		if err != nil {
			t.svcCtx.Logger.Error("Scheduler create task failed", zap.String("id", task.ID),
				zap.String("name", task.Name), zap.Error(err))
			failedTasks = append(failedTasks, task.ID)

			// 回滚：从数据库删除失败的任务
			if delErr := t.svcCtx.DAO.Task.DeleteByID(ctx, task.ID); delErr != nil {
				t.svcCtx.Logger.Error("Rollback database task failed",
					zap.String("id", task.ID),
					zap.Error(delErr))
			}
		} else {
			t.svcCtx.Logger.Info("Scheduler create task success",
				zap.String("id", taskID),
				zap.String("name", task.Name))
		}
	}

	if len(failedTasks) > 0 {
		return fmt.Errorf("Scheduler create task failed: %v", failedTasks)
	}

	t.svcCtx.Logger.Info("Task batch create success", zap.Int("count", len(req)))
	return nil
}

// validateSchedulerTask 验证调度器任务参数
func (t *Task) validateSchedulerTask(task *models.Task) error {
	switch task.Type {
	case models.TaskType_CRON:
		if task.CronExpression == "" {
			return fmt.Errorf("cron任务需要cron表达式")
		}
	case models.TaskType_RECURRING:
		if task.CronExpression == "" {
			return fmt.Errorf("周期任务参数无效")
		}
	case models.TaskType_PERIODIC:
		if task.CronExpression == "" || task.StartTime <= 0 || task.EndTime <= 0 || task.StartTime >= task.EndTime {
			return fmt.Errorf("周期任务参数无效")
		}
	}
	return nil
}
