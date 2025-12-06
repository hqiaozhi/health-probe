package task

import (
	"fmt"
	"health-probe/internal/models"

	"github.com/gin-gonic/gin"
)

func (t *Task) Update(c *gin.Context) {
	// 获取ID
	id := c.Param("id")
	if id == "" {
		t.svcCtx.Resp.RESP_BAD_REQUEST(c, "ID cannot be empty")
		return
	}

	var req *models.Task
	if err := c.ShouldBindJSON(&req); err != nil {
		t.svcCtx.Resp.RESP_ERROR(c, "Invalid request parameters")
		return
	}
	req.ID = id
	fmt.Println("=================\n", req, "\n=================")
	err := t.task.Update(c, req)
	if err != nil {
		t.svcCtx.Resp.RESP_ERROR(c, err.Error())
		return
	}
	t.svcCtx.Resp.RESP_OK(c)
}
