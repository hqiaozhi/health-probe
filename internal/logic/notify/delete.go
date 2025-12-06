package notify

import "context"

func (n *Notify) DeleteByIDList(ctx context.Context, req []string) error {
	return n.svcCtx.DAO.EmailLog.BatchDeleteByIDs(ctx, req)
}

// 通过id删除邮件记录
func (n *Notify) DeleteByID(ctx context.Context, req string) error {
	return n.svcCtx.DAO.EmailLog.DeleteByID(ctx, req)
}
