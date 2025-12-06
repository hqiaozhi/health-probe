package task

import (
	"context"
	"fmt"

	"go.uber.org/zap"
)

func (t *Task) BatchDelete(ctx context.Context, req []string) error {
	// 先批量删除调度器中的任务（容忍部分任务不存在）
	var failedIDs []string
	for _, id := range req {
		err := t.svcCtx.Scheduler.DeleteTask(id)
		if err != nil {
			// 如果是"task not found"错误，记录警告但不中断流程
			if err.Error() == fmt.Sprintf("task %s not found", id) {
				t.svcCtx.Logger.Warn("调度器中任务不存在，跳过删除", zap.String("id", id))
			} else {
				// 其他错误记录并收集失败的任务ID
				t.svcCtx.Logger.Error("从调度器删除任务失败", zap.String("id", id), zap.Error(err))
				failedIDs = append(failedIDs, id)
			}
		} else {
			t.svcCtx.Logger.Info("从调度器删除任务成功", zap.String("id", id))
		}
	}

	// 如果调度器删除有失败，且不是"not found"错误，则返回错误
	if len(failedIDs) > 0 {
		return fmt.Errorf("以下任务从调度器删除失败: %v", failedIDs)
	}

	// 删除数据库中的任务记录
	return t.svcCtx.DAO.Task.BatchDeleteByIDs(ctx, req)
}

// 通过id删除任务
func (t *Task) DeleteByID(ctx context.Context, id string) error {
	// 先从调度器删除任务
	err := t.svcCtx.Scheduler.DeleteTask(id)
	if err != nil {
		// 如果是"task not found"错误，记录警告但不中断流程
		if err.Error() == fmt.Sprintf("task %s not found", id) {
			t.svcCtx.Logger.Warn("调度器中任务不存在，跳过删除", zap.String("id", id))
		} else {
			// 其他错误记录并返回
			t.svcCtx.Logger.Error("从调度器删除任务失败", zap.String("id", id), zap.Error(err))
			return err
		}
	} else {
		t.svcCtx.Logger.Info("从调度器删除任务成功", zap.String("id", id))
	}

	// 从数据库删除任务记录
	err = t.svcCtx.DAO.Task.DeleteByID(ctx, id)
	if err != nil {
		t.svcCtx.Logger.Error("从数据库删除任务失败", zap.String("id", id), zap.Error(err))
		return err
	}

	t.svcCtx.Logger.Info("从数据库删除任务成功", zap.String("id", id))

	return nil
}
