package task

import "github.com/gin-gonic/gin"

func (t *Task) DeleteByIDList(c *gin.Context) {
	// 判断请求内容是否为空
	if c.Request.ContentLength == 0 {
		t.svcCtx.Resp.RESP_BAD_REQUEST(c, "The request cannot be empty")
		return
	}
	type request struct {
		Ids []string `json:"ids" form:"ids" binding:"required"`
	}
	var req request
	if err := c.ShouldBindJSON(&req); err != nil {
		t.svcCtx.Resp.RESP_ERROR(c, "Invalid request parameters")
		return
	}
	if len(req.Ids) == 0 {
		t.svcCtx.Resp.RESP_BAD_REQUEST(c, "Ids cannot be empty")
		return
	}
	if err := t.task.BatchDelete(c, req.Ids); err != nil {
		t.svcCtx.Resp.RESP_ERROR(c, err.Error())
		return
	}
	t.svcCtx.Resp.RESP_OK(c)
}

func (t *Task) DeleteByID(c *gin.Context) {
	// 获取ID
	id := c.Param("id")
	if id == "" {
		t.svcCtx.Resp.RESP_BAD_REQUEST(c, "ID cannot be empty")
		return
	}

	if err := t.task.DeleteByID(c, id); err != nil {
		t.svcCtx.Resp.RESP_ERROR(c, err.Error())
		return
	}
	t.svcCtx.Resp.RESP_OK(c)
}
