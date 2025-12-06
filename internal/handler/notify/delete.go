package notify

import (
	"github.com/gin-gonic/gin"
)

func (n *Notify) DeleteByIDList(c *gin.Context) {
	// 判断请求内容是否为空
	if c.Request.ContentLength == 0 {
		n.svcCtx.Resp.RESP_BAD_REQUEST(c, "The request cannot be empty")
		return
	}

	var req struct {
		Ids []string `json:"ids" form:"ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		n.svcCtx.Resp.RESP_ERROR(c, "Invalid request parameters")
		return
	}
	// 判断IDs是否为空
	if len(req.Ids) == 0 {
		n.svcCtx.Resp.RESP_BAD_REQUEST(c, "Ids cannot be empty")
		return
	}

	if err := n.notify.DeleteByIDList(c, req.Ids); err != nil {
		n.svcCtx.Resp.RESP_ERROR(c, err.Error())
		return
	}

	n.svcCtx.Resp.RESP_OK(c)
}

// 通过id删除
func (n *Notify) DeleteByID(c *gin.Context) {
	// 获取ID
	id := c.Param("id")
	if id == "" {
		n.svcCtx.Resp.RESP_BAD_REQUEST(c, "ID cannot be empty")
		return
	}

	if err := n.notify.DeleteByID(c, id); err != nil {
		n.svcCtx.Resp.RESP_ERROR(c, "delete fail")
	}
	n.svcCtx.Resp.RESP_OK(c)
}
