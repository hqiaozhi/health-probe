package notify

import (
	"context"

	"health-probe/internal/models"
)

func (n *Notify) Query(ctx context.Context, req models.EmailLogQueryCondition) (resp []*models.EmailLog, total int64, err error) {
	resp, total, err = n.svcCtx.DAO.EmailLog.ListByCondition(ctx, req)
	return
}
