package alert

import (
	"fmt"
	"html/template"
	"strings"
)

// 导入或直接定义 ProbeType 枚举（如果已在其他包定义，可删除此处，保留导入）
type ProbeType int32

const (
	ProbeType_HTTP ProbeType = 0
	ProbeType_TCP  ProbeType = 1
	ProbeType_DNS  ProbeType = 2
	ProbeType_PING ProbeType = 3
)

// Enum value maps for ProbeType.（保留原有映射表）
var (
	ProbeType_name = map[int32]string{
		0: "HTTP",
		1: "TCP",
		2: "DNS",
		3: "PING",
	}
	ProbeType_value = map[string]int32{
		"HTTP": 0,
		"TCP":  1,
		"DNS":  2,
		"PING": 3,
	}
)

// probeTypeToCN 枚举对应的中文名称映射（用于友好显示）
var probeTypeToCN = map[ProbeType]string{
	ProbeType_HTTP: "HTTP探测",
	ProbeType_TCP:  "TCP端口探测",
	ProbeType_DNS:  "DNS解析探测",
	ProbeType_PING: "PING连通性探测",
}

// 基础 HTML 模板常量（保持原有样式）
const baseHTMLTemplate = `
<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.Title}}</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; font-family: "Microsoft YaHei", Arial, sans-serif; }
        .alert-container { max-width: 800px; margin: 20px auto; padding: 24px; border-radius: 8px; box-shadow: 0 2px 12px rgba(0,0,0,0.1); }
        .alert-header { padding-bottom: 16px; margin-bottom: 16px; border-bottom: 1px solid #eee; }
        .alert-title { font-size: 20px; font-weight: 600; margin-bottom: 8px; }
        .alert-status { display: inline-block; padding: 4px 12px; border-radius: 20px; font-size: 14px; font-weight: 500; }
        .status-success { background-color: #e6f7ef; color: #008641; }
        .status-warning { background-color: #fff7e6; color: #fa8c16; }
        .alert-content { font-size: 16px; line-height: 1.8; color: #333; }
        .alert-content p { margin-bottom: 12px; }
        .alert-label { display: inline-block; width: 100px; font-weight: 500; color: #666; }
        .alert-value { color: #1f2937; font-weight: 500; }
        .alert-footer { margin-top: 24px; padding-top: 16px; border-top: 1px solid #eee; font-size: 14px; color: #999; }
    </style>
</head>
<body>
    <div class="alert-container">
        <div class="alert-header">
            <h2 class="alert-title">{{.Title}}</h2>
            <span class="alert-status {{.StatusClass}}">{{.StatusText}}</span>
        </div>
        <div class="alert-content">
            {{range .Details}}
            <p><span class="alert-label">{{.Key}}：</span><span class="alert-value">{{.Value}}</span></p>
            {{end}}
        </div>
        <div class="alert-footer">
            告警时间：{{.AlertTime}}
        </div>
    </div>
</body>
</html>
`

// 模板数据结构（统一复用）
type templateData struct {
	Title       string       // 告警标题
	StatusText  string       // 状态文本（如「恢复正常」「连续失败」）
	StatusClass string       // 状态样式类（对应 CSS）
	Details     []detailItem // 详细信息列表
	AlertTime   string       // 告警时间（格式化字符串）
}

// 详细信息项
type detailItem struct {
	Key   string // 字段名（如「任务名称」）
	Value string // 字段值
}

// GenerateRecoveryAlertHTML 生成「任务恢复」告警的 HTML 字符串（支持 ProbeType 枚举）
// 参数说明：
// - taskName: 任务名称
// - probeType: 任务类型（ProbeType 枚举）
// - target: 探测目标（如 IP、URL、端口）
// - taskID: 任务 ID
// - durationMin: 持续时间（分钟）
// - alertTime: 告警时间（格式化字符串，如 "2025-11-29 14:30:00"）
// 返回：HTML 字符串 + 错误（模板渲染失败时返回）
func GenerateRecoveryAlertHTML(
	taskName string,
	probeType ProbeType,
	target string,
	taskID string,
	durationMin int,
	alertTime string,
) (string, error) {
	data := templateData{
		Title:       "【任务恢复通知】",
		StatusText:  "恢复正常",
		StatusClass: "status-success",
		AlertTime:   alertTime,
		Details: []detailItem{
			{Key: "任务名称", Value: taskName},
			{Key: "任务类型", Value: getProbeTypeCN(probeType)}, // 显示中文名称
			{Key: "探测目标", Value: target},
			{Key: "任务ID", Value: taskID},
			{Key: "持续时间", Value: sprintf("%d 分钟", durationMin)},
		},
	}
	return renderTemplate(data)
}

// GenerateContinuousFailAlertHTML 生成「任务连续失败」告警的 HTML 字符串（支持 ProbeType 枚举）
// 参数说明：
// - taskName: 任务名称
// - probeType: 任务类型（ProbeType 枚举）
// - target: 探测目标（如 IP、URL、端口）
// - taskID: 任务 ID
// - durationMin: 统计时长（分钟）
// - failCount: 连续失败次数
// - alertTime: 告警时间（格式化字符串，如 "2025-11-29 14:30:00"）
// 返回：HTML 字符串 + 错误（模板渲染失败时返回）
func GenerateContinuousFailAlertHTML(
	taskName string,
	probeType ProbeType,
	target string,
	taskID string,
	durationMin int,
	failCount int,
	alertTime string,
) (string, error) {
	data := templateData{
		Title:       "【任务失败告警】",
		StatusText:  "连续失败",
		StatusClass: "status-warning",
		AlertTime:   alertTime,
		Details: []detailItem{
			{Key: "任务名称", Value: taskName},
			{Key: "任务类型", Value: getProbeTypeCN(probeType)}, // 显示中文名称
			{Key: "探测目标", Value: target},
			{Key: "任务ID", Value: taskID},
			{Key: "统计时长", Value: sprintf("%d 分钟", durationMin)},
			{Key: "已连续失败", Value: sprintf("%d 次", failCount)},
		},
	}
	return renderTemplate(data)
}

// 私有工具函数：渲染模板
func renderTemplate(data templateData) (string, error) {
	tpl, err := template.New("alert").Parse(baseHTMLTemplate)
	if err != nil {
		return "", err
	}

	var buf strings.Builder
	if err := tpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// 私有工具函数：将 ProbeType 枚举转为中文名称（核心优化）
func getProbeTypeCN(pt ProbeType) string {
	// 优先从中文映射表获取
	if cnName, ok := probeTypeToCN[pt]; ok {
		return cnName
	}
	// 降级：如果枚举值未定义中文，显示原始英文名称（如 ProbeType_name 中的值）
	if enName, ok := ProbeType_name[int32(pt)]; ok {
		return enName + "探测"
	}
	// 兜底：未知类型
	return "未知探测类型"
}

// 私有工具函数：简化字符串格式化
func sprintf(format string, args ...interface{}) string {
	return fmt.Sprintf(format, args...)
}
