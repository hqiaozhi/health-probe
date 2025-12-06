package task

import (
	"fmt"
	"health-probe/internal/models"

	"github.com/gin-gonic/gin"
)

// 0: "IMMEDIATE",
// 1: "CRON",
// 2: "RECURRING",
// 3: "PERIODIC",

// 0: "SCHEDULED",
// 1: "PAUSED",
// 2: "STOPPED",

func (t *Task) Create(c *gin.Context) {
	// 判断请求内容是否为空
	if c.Request.ContentLength == 0 {
		t.svcCtx.Resp.RESP_BAD_REQUEST(c, "The request cannot be empty")
		return
	}
	var req struct {
		Items []*models.Task `json:"items" binding:"required,min=1,max=50"`
	}
	// 单独校验

	if err := c.ShouldBind(&req); err != nil {
		t.svcCtx.Resp.RESP_BAD_REQUEST(c, "Invalid request parameters")
		return
	}
	if len(req.Items) == 0 {
		t.svcCtx.Resp.RESP_BAD_REQUEST(c, "The request cannot be empty")
		return
	}
	for _, item := range req.Items {
		fmt.Println(item.CronExpression)
	}
	if err := t.task.BatchCreate(c, req.Items); err != nil {
		t.svcCtx.Resp.RESP_ERROR(c, err.Error())
		return
	}
	t.svcCtx.Resp.RESP_OK(c)
}
