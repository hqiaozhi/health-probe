package task

import (
	logic "health-probe/internal/logic/task"
	"health-probe/internal/svc"

	"github.com/gin-gonic/gin"
)

type ITask interface {
	Query(c *gin.Context)
	Create(c *gin.Context)
	DeleteByID(c *gin.Context)
	DeleteByIDList(c *gin.Context)
	Update(c *gin.Context)
	Control(c *gin.Context)
}

type Task struct {
	svcCtx *svc.ServiceContext
	task   *logic.Task
}

func New(svcCtx *svc.ServiceContext) ITask {
	return &Task{
		svcCtx: svcCtx,
		task:   logic.New(svcCtx),
	}
}
