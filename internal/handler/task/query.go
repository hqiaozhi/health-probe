package task

import (
	"health-probe/internal/models"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

func (t *Task) Query(c *gin.Context) {
	// 手动解析查询参数，避免ShouldBindQuery对枚举类型的处理问题
	var QueryCondition struct {
		IDs       []string           `json:"ids"`
		Name      *string            `json:"name"`
		TaskType  *models.TaskType   `json:"task_type"`
		Status    *models.TaskStatus `json:"status"`
		ProbeType *models.ProbeType  `json:"probe_type"`
		Target    *string            `json:"target"`
		Page      int                `json:"page"`
		PageSize  int                `json:"page_size"`
		SortField string             `json:"sort_field"`
		SortOrder string             `json:"sort_order"`
	}

	// 设置默认分页参数
	if QueryCondition.Page <= 0 {
		QueryCondition.Page = 1
	}
	if QueryCondition.PageSize <= 0 {
		QueryCondition.PageSize = 20
	}

	// 手动解析IDs参数
	if ids := c.Query("ids"); ids != "" {
		QueryCondition.IDs = strings.Split(ids, ",")
	}
	// 手动解析Name参数
	if name := c.Query("name"); name != "" {
		QueryCondition.Name = &name
	}

	// 手动解析枚举类型参数
	if taskTypeStr := c.Query("task_type"); taskTypeStr != "" {
		if val, err := strconv.Atoi(taskTypeStr); err == nil {
			taskType := models.TaskType(val)
			QueryCondition.TaskType = &taskType
		}
	}

	if statusStr := c.Query("status"); statusStr != "" {
		if val, err := strconv.Atoi(statusStr); err == nil {
			status := models.TaskStatus(val)
			QueryCondition.Status = &status
		}
	}

	if probeTypeStr := c.Query("probe_type"); probeTypeStr != "" {
		if val, err := strconv.Atoi(probeTypeStr); err == nil {
			probeType := models.ProbeType(val)
			QueryCondition.ProbeType = &probeType
		}
	}

	// 解析其他参数
	if target := c.Query("target"); target != "" {
		QueryCondition.Target = &target
	}

	// 解析分页和排序参数
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

	if sortField := c.Query("sort_field"); sortField != "" {
		if sortField != "" {
			QueryCondition.SortField = sortField
		} else {
			QueryCondition.SortField = "created_at"
		}
		if sortOrder := c.Query("sort_order"); sortOrder != "" {
			QueryCondition.SortOrder = sortOrder
		} else {
			QueryCondition.SortOrder = "asc"
		}
	}

	// 添加调试信息
	t.svcCtx.Logger.Info("Received query parameters", zap.Any("params", c.Request.URL.Query()))
	t.svcCtx.Logger.Info("Parsed QueryCondition", zap.Any("QueryCondition", QueryCondition))

	// 构建查询条件
	req := models.TaskQueryCondition{
		IDs:            QueryCondition.IDs,
		Name:           QueryCondition.Name,
		TaskType:       QueryCondition.TaskType,
		Status:         QueryCondition.Status,
		ProbeType:      QueryCondition.ProbeType,
		Target:         QueryCondition.Target,
		Page:           QueryCondition.Page,
		PageSize:       QueryCondition.PageSize,
		SortField:      QueryCondition.SortField,
		SortOrder:      QueryCondition.SortOrder,
		WithSoftDelete: false, // 默认不包含软删除数据
	}

	// 根据IDs区分批量查询和分页查询
	if len(req.IDs) > 0 {
		// 限制最大批量查询数量（避免 URL 过长+数据库压力）
		if len(req.IDs) > 100 {
			t.svcCtx.Resp.RESP_ERROR(c, "最多支持100个ID批量查询")
			return
		}
		// 校验每个 ID 格式
		for _, id := range req.IDs {
			if _, err := uuid.Parse(id); err != nil {
				t.svcCtx.Resp.RESP_ERROR(c, "ID格式无效："+id)
				return
			}
		}
		taskList, err := t.task.BatchQuery(c, req.IDs)
		if err != nil {
			t.svcCtx.Resp.RESP_ERROR(c, err.Error())
			return
		}
		var data struct {
			Total int64          `json:"total"`
			Items []*models.Task `json:"items"`
		}
		data.Items = taskList
		data.Total = int64(len(taskList))
		t.svcCtx.Resp.RESP_DATA(c, data)
	} else {
		taskList, total, err := t.task.Query(c, req)
		if err != nil {
			t.svcCtx.Resp.RESP_ERROR(c, err.Error())
			return
		}
		var data struct {
			Total int64          `json:"total"`
			Items []*models.Task `json:"items"`
		}
		data.Items = taskList
		data.Total = total
		t.svcCtx.Resp.RESP_DATA(c, data)
	}
}
