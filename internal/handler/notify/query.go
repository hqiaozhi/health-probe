package notify

import (
	"health-probe/internal/models"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func (n *Notify) Query(c *gin.Context) {
	// 手动解析查询参数，避免ShouldBindQuery对枚举类型的处理问题
	var QueryCondition struct {
		TaskID     *string                `json:"task_id" form:"task_id"`
		TaskName   *string                `json:"task_name" form:"task_name"`
		ProbeType  *models.ProbeType      `json:"probe_type" form:"probe_type"`
		Target     *string                `json:"target" form:"target"`
		AlertType  *models.AlertType      `json:"alert_type" form:"alert_type"`
		SendStatus *models.EmailLogStatus `json:"send_status" form:"send_status"`
		Is_Read    *bool                  `json:"is_read" form:"is_read"`
		Page       int                    `json:"page" form:"page"`
		PageSize   int                    `json:"page_size" form:"page_size"`
	}

	// 设置默认分页参数
	if QueryCondition.Page <= 0 {
		QueryCondition.Page = 1
	}
	if QueryCondition.PageSize <= 0 {
		QueryCondition.PageSize = 10
	}

	// 手动解析任务名称参数
	if taskName := c.Query("task_name"); taskName != "" {
		QueryCondition.TaskName = &taskName
	}

	// 手动解析枚举类型参数
	if probeTypeStr := c.Query("probe_type"); probeTypeStr != "" {
		if val, err := strconv.Atoi(probeTypeStr); err == nil {
			probeType := models.ProbeType(val)
			QueryCondition.ProbeType = &probeType
		}
	}

	if alertTypeStr := c.Query("alert_type"); alertTypeStr != "" {
		if val, err := strconv.Atoi(alertTypeStr); err == nil {
			alertType := models.AlertType(val)
			QueryCondition.AlertType = &alertType
		}
	}

	if sendStatusStr := c.Query("send_status"); sendStatusStr != "" {
		if val, err := strconv.Atoi(sendStatusStr); err == nil {
			sendStatus := models.EmailLogStatus(val)
			QueryCondition.SendStatus = &sendStatus
		}
	}

	// 解析其他参数
	if taskID := c.Query("task_id"); taskID != "" {
		QueryCondition.TaskID = &taskID
	}

	if target := c.Query("target"); target != "" {
		QueryCondition.Target = &target
	}

	// 解析布尔类型参数
	if isReadStr := c.Query("is_read"); isReadStr != "" {
		if isRead, err := strconv.ParseBool(isReadStr); err == nil {
			QueryCondition.Is_Read = &isRead
		}
	}

	// 解析分页参数
	if pageStr := c.Query("page"); pageStr != "" {
		if page, err := strconv.Atoi(pageStr); err == nil && page > 0 {
			QueryCondition.Page = page
		}
	}

	if pageSizeStr := c.Query("page_size"); pageSizeStr != "" {
		if pageSize, err := strconv.Atoi(pageSizeStr); err == nil && pageSize > 0 {
			QueryCondition.PageSize = pageSize
		}
	}

	// 添加调试信息
	n.svcCtx.Logger.Info("Received query parameters", zap.Any("params", c.Request.URL.Query()))
	n.svcCtx.Logger.Info("Parsed PageQuery", zap.Any("PageQuery", QueryCondition))

	// 构建查询条件
	req := models.EmailLogQueryCondition{
		TaskID:     QueryCondition.TaskID,
		TaskName:   QueryCondition.TaskName,
		ProbeType:  QueryCondition.ProbeType,
		Target:     QueryCondition.Target,
		AlertType:  QueryCondition.AlertType,
		SendStatus: QueryCondition.SendStatus,
		Is_Read:    QueryCondition.Is_Read,
		Page:       QueryCondition.Page,
		PageSize:   QueryCondition.PageSize,
	}

	resp, total, err := n.notify.Query(c, req)
	if err != nil {
		n.svcCtx.Logger.Error("Query email log failed", zap.Error(err))
		return
	}
	data := map[string]interface{}{
		"total": total,
		"items": resp,
	}

	// 成功响应
	n.svcCtx.Resp.RESP_DATA(c, data)
}
