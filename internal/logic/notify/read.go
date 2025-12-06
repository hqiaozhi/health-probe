package notify

import "context"

func (n *Notify) UpdateReadStatus(ctx context.Context, ID string) error {
	return n.svcCtx.DAO.EmailLog.UpdateReadStatus(ctx, ID)
}
