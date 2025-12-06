package task

import (
	"strings"

	"github.com/gin-gonic/gin"
)

func (t *Task) Control(c *gin.Context) {
	var req struct {
		IDs    string `json:"ids" form:"ids" binding:"required"`
		Status string `json:"status" form:"status" binding:"required"`
	}
	if err := c.ShouldBindQuery(&req); err != nil {
		t.svcCtx.Resp.RESP_ERROR(c, "Invalid request parameters")
		return
	}
	idList := strings.Split(req.IDs, ",")

	err := t.task.Control(c, idList, req.Status)
	if err != nil {
		t.svcCtx.Resp.RESP_ERROR(c, err.Error())
		return
	}

	// 成功响应
	t.svcCtx.Resp.RESP_OK(c)
}
