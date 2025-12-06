package task

import (
	"context"
	"health-probe/internal/models"
)

func (t *Task) Query(ctx context.Context, req models.TaskQueryCondition) ([]*models.Task, int64, error) {
	return t.svcCtx.DAO.Task.ListByCondition(ctx, req)
}

func (t *Task) BatchQuery(ctx context.Context, ID []string) ([]*models.Task, error) {
	return t.svcCtx.DAO.Task.BatchGetByIDs(ctx, ID)
}
