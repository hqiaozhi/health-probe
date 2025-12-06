package task

import (
	"context"
	"health-probe/internal/core/scheduler"
	"health-probe/internal/models"

	"go.uber.org/zap"
)

func (t *Task) Update(ctx context.Context, task *models.Task) error {
	// 先更新数据库，再更新调度器，确保数据一致性
	if err := t.svcCtx.DAO.Task.Update(ctx, task); err != nil {
		return err
	}
	t.svcCtx.Logger.Info("update database task success", zap.String("taskID", task.ID))

	// 查询出数据库的内容更新到调度器中
	Newtask, err := t.svcCtx.DAO.Task.GetByID(ctx, task.ID)
	if err != nil {
		return err
	}
	err = t.svcCtx.Scheduler.UpdateTask(ctx, &scheduler.Task{
		ID:   Newtask.ID,
		Name: Newtask.Name,
		Type: scheduler.TaskType(Newtask.Type),
		ProbeConfig: &scheduler.ProbeConfig{
			Type:                  scheduler.ProbeType(Newtask.ProbeConfig.Type),
			Target:                Newtask.ProbeConfig.Target,
			Tls:                   Newtask.ProbeConfig.Tls,
			Port:                  Newtask.ProbeConfig.Port,
			Timeout:               Newtask.ProbeConfig.Timeout,
			FailureWindowDuration: Newtask.ProbeConfig.FailureWindowDuration,
			SuccessWindowDuration: Newtask.ProbeConfig.SuccessWindowDuration,
			FailureThreshold:      Newtask.ProbeConfig.FailureThreshold,
			AlertEmailRecipients:  Newtask.ProbeConfig.AlertEmailRecipients,
		},
		CronExpression: Newtask.CronExpression,
		StartTime:      Newtask.StartTime,
		EndTime:        Newtask.EndTime,
		Status:         scheduler.TaskStatus(Newtask.Status),
	})
	if err != nil {
		t.svcCtx.Logger.Error("update scheduler task failed",
			zap.String("taskID", Newtask.ID), zap.Error(err))
	}
	t.svcCtx.Logger.Info("update scheduler task success", zap.String("taskID", Newtask.ID))
	return nil
}
