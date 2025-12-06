package notify

import "health-probe/internal/svc"

type Notify struct {
	svcCtx *svc.ServiceContext
}

func New(svcCtx *svc.ServiceContext) *Notify {
	return &Notify{
		svcCtx: svcCtx,
	}
}
