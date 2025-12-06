package notify

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func (n *Notify) UpdateReadStatus(c *gin.Context) {
	// 获取ID
	id := c.Param("id")
	if id == "" {
		n.svcCtx.Resp.RESP_BAD_REQUEST(c, "ID cannot be empty")
		return
	}

	err := n.notify.UpdateReadStatus(c, id)
	if err != nil {
		n.svcCtx.Logger.Error("Update read status failed", zap.Error(err))
		n.svcCtx.Resp.RESP_ERROR(c, "Update read status failed")
		return
	}

	// 响应成功
	n.svcCtx.Resp.RESP_OK(c)
}
