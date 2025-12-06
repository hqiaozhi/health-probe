package notify

import (
	"health-probe/internal/logic/notify"
	"health-probe/internal/svc"

	"github.com/gin-gonic/gin"
)

type INotify interface {
	Query(c *gin.Context)
	DeleteByIDList(c *gin.Context)
	DeleteByID(c *gin.Context)
	UpdateReadStatus(c *gin.Context)
}

type Notify struct {
	svcCtx *svc.ServiceContext
	notify *notify.Notify
}

func New(svcCtx *svc.ServiceContext) INotify {
	return &Notify{
		svcCtx: svcCtx,
		notify: notify.New(svcCtx),
	}
}
