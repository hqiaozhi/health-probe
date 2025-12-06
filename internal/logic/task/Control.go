package task

import (
	"context"
	"health-probe/internal/core/scheduler"

	"go.uber.org/zap"
)

func (t *Task) Control(ctx context.Context, ids []string, status string) error {
	if status == "sched" {
		err := t.svcCtx.DAO.Task.BatchUpdateStatus(ctx, ids, 0)
		if err != nil {
			return err
		}
		t.svcCtx.Logger.Info("Database task resume success", zap.Strings("ids", ids))
		for _, id := range ids {
			task, err := t.svcCtx.DAO.Task.GetByID(ctx, id)
			if err != nil {
				return err
			}
			_, err = t.svcCtx.Scheduler.CreateTask(ctx, &scheduler.Task{
				ID:   task.ID,
				Name: task.Name,
				Type: scheduler.TaskType(task.Type),
				ProbeConfig: &scheduler.ProbeConfig{
					Type:                  scheduler.ProbeType(task.ProbeConfig.Type),
					Target:                task.ProbeConfig.Target,
					Tls:                   task.ProbeConfig.Tls,
					Port:                  task.ProbeConfig.Port,
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
			if err != nil {
				return err
			}
		}
		t.svcCtx.Logger.Info("Scheduler task create success", zap.Strings("ids", ids))
	} else if status == "stop" {
		err := t.svcCtx.DAO.Task.BatchUpdateStatus(ctx, ids, 1)
		if err != nil {
			return err
		}
		t.svcCtx.Logger.Info("Database task stop success", zap.Strings("ids", ids))
		for _, id := range ids {
			if err := t.svcCtx.Scheduler.DeleteTask(id); err != nil {
				return err
			}
		}
		t.svcCtx.Logger.Info("Scheduler Task delete success", zap.Strings("ids", ids))
	}
	return nil
}
