package scheduler

import (
	"context"
	"health-probe/internal/core/metrics"
	"health-probe/internal/core/rule"
	"health-probe/internal/core/tools"
	dns "health-probe/internal/core/tools/dns"
	http "health-probe/internal/core/tools/http"
	"health-probe/internal/core/tools/ping"
	tcp "health-probe/internal/core/tools/tcp"
	"health-probe/internal/models"
	"strconv"
	"time"

	"go.uber.org/zap"
)

// Executor 任务执行器，负责调用外部探测服务
type Executor struct {
	tools   tools.Tools
	rule    *rule.MemoryAlertStatService
	metrics *metrics.Metrics
	logger  *zap.Logger
}

// NewExecutor 创建执行器实例
func NewExecutor(tools tools.Tools, rule *rule.MemoryAlertStatService, metrics *metrics.Metrics, logger *zap.Logger) *Executor {
	return &Executor{
		tools:   tools,
		rule:    rule,
		metrics: metrics,
		logger:  logger,
	}
}

// Execute 执行任务，调用对应的探测服务
func (e *Executor) Execute(ctx context.Context, task *Task) {
	e.logger.Info("Starting execution of task", zap.String("task", task.ID),
		zap.String("type", string(task.ProbeConfig.Type)))
	probeParam := rule.ProbeParam{
		TaskID:                task.ID,
		TaskName:              task.Name,
		Target:                task.ProbeConfig.Target,
		ProbeType:             models.ProbeType(task.ProbeConfig.Type),
		CurrentTimeMs:         time.Now().UnixMilli(),
		FailureWindowDuration: task.ProbeConfig.FailureWindowDuration * 1000 * 60,
		SuccessWindowDuration: task.ProbeConfig.SuccessWindowDuration * 1000 * 60,
		FailureThreshold:      task.ProbeConfig.FailureThreshold,
		AlertEmailRecipients:  task.ProbeConfig.AlertEmailRecipients,
	}

	// 根据探测类型调用相应的探测服务
	switch task.ProbeConfig.Type {
	case ProbeType_HTTP:
		result := e.handleHttpProbe(ctx, task)
		probeParam.IsSuccess = result.Success
		if result.Success {
			e.metrics.HttpSuccess.WithLabelValues(task.ProbeConfig.Target).Set(1)
		} else {
			e.metrics.HttpSuccess.WithLabelValues(task.ProbeConfig.Target).Set(0)
		}
		e.metrics.HttpStatusCode.WithLabelValues(task.ProbeConfig.Target).Set(float64(result.HttpStatus))
		e.metrics.HttpDuration.WithLabelValues(task.ProbeConfig.Target).Observe(float64(result.Duration))
		e.rule.Probe(probeParam)
	case ProbeType_TCP:
		result := e.handleTcpProbe(ctx, task)
		probeParam.IsSuccess = result.Success
		if result.Success {
			e.metrics.TcpSuccess.WithLabelValues(task.ProbeConfig.Target).Set(1)
		} else {
			e.metrics.TcpSuccess.WithLabelValues(task.ProbeConfig.Target).Set(0)
		}
		e.metrics.TcpDuration.WithLabelValues(task.ProbeConfig.Target).Observe(float64(result.Duration))
		e.rule.Probe(probeParam)
	case ProbeType_DNS:
		result := e.handleDnsProbe(ctx, task)
		probeParam.IsSuccess = result.Success
		if result.Success {
			e.metrics.DnsSuccess.WithLabelValues(task.ProbeConfig.Target).Set(1)
		} else {
			e.metrics.DnsSuccess.WithLabelValues(task.ProbeConfig.Target).Set(0)
		}
		e.metrics.DnsDuration.WithLabelValues(task.ProbeConfig.Target).Observe(float64(result.Duration))
		e.rule.Probe(probeParam)
	case ProbeType_PING:
		result := e.handlePingProbe(ctx, task)
		probeParam.IsSuccess = result.Success
		if result.Success {
			e.metrics.IcmpSuccess.WithLabelValues(task.ProbeConfig.Target).Set(1)
		} else {
			e.metrics.IcmpSuccess.WithLabelValues(task.ProbeConfig.Target).Set(0)
		}
		e.metrics.IcmpDuration.WithLabelValues(task.ProbeConfig.Target).Observe(float64(result.Duration))
		e.rule.Probe(probeParam)
	default:
		e.logger.Error("Unsupported probe type", zap.String("task", task.ID), zap.String("type", string(task.ProbeConfig.Type)))
	}
}

// 处理HTTP探测
func (e *Executor) handleHttpProbe(ctx context.Context, task *Task) (result TaskResult) {
	timeout := task.ProbeConfig.Timeout
	if timeout <= 0 {
		timeout = 5 // 默认5秒
	}

	e.logger.Info("Handling HTTP probe", zap.String("task", task.ID),
		zap.String("target", task.ProbeConfig.Target), zap.Int("timeout", int(timeout)),
		zap.Bool("tls", task.ProbeConfig.Tls))

	// 创建带超时的上下文
	ctxWithTimeout, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()

	// 调用外部HTTP探测服务
	req := &http.Request{
		Target:  task.ProbeConfig.Target,
		Timeout: timeout,
		Method:  "GET",
		Path:    "/",
		Tls:     task.ProbeConfig.Tls,
	}

	e.logger.Info("Sending HttpProbe request", zap.Any("request", req))

	// 添加调试日志，查看连接状态
	e.logger.Info("About to call HttpProbe with context deadline", zap.Error(ctxWithTimeout.Err()))

	resp, err := e.tools.HTTP.Probe(ctxWithTimeout, req)

	if err != nil {
		result.Error = err.Error()
		e.logger.Error("HttpProbe failed", zap.String("task", task.ID), zap.Error(err))
		// 添加更多错误详情
		if ctxWithTimeout.Err() != nil {
			e.logger.Error("Context error", zap.Error(ctxWithTimeout.Err()))
		}
		return
	}

	result.Success = resp.Success
	result.HttpStatus = int32(resp.Code)
	result.Duration = resp.Duration
	// 添加探测类型
	result.Type = "http"

	e.logger.Info("HttpProbe succeeded", zap.String("task", task.ID), zap.String("target", task.ProbeConfig.Target), zap.Int("status", resp.Code))
	return result
}

// 处理TCP探测
func (e *Executor) handleTcpProbe(ctx context.Context, task *Task) (result TaskResult) {
	timeout := task.ProbeConfig.Timeout
	if timeout <= 0 {
		timeout = 5 // 默认5秒
	}

	e.logger.Info("Handling TCP probe", zap.String("task", task.ID),
		zap.String("target", task.ProbeConfig.Target), zap.Int("timeout", int(timeout)))

	// 创建带超时的上下文
	ctxWithTimeout, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()

	// 调用外部TCP探测服务
	req := &tcp.Request{
		Target:  task.ProbeConfig.Target,
		Port:    strconv.FormatInt(task.ProbeConfig.Port, 10),
		Timeout: timeout,
	}

	e.logger.Info("Sending TcpProbe request", zap.Any("request", req))

	resp, err := e.tools.TCP.Probe(ctxWithTimeout, req)

	if err != nil {
		result.Error = err.Error()
		e.logger.Error("TcpProbe failed", zap.String("task", task.ID), zap.Error(err))
		return
	}

	result.Success = resp.Success
	result.Duration = resp.Duration
	result.Type = "tcp"

	e.logger.Info("TcpProbe succeeded", zap.String("task", task.ID), zap.String("target", task.ProbeConfig.Target), zap.Bool("status", resp.Success))
	return result
}

// 处理DNS探测
func (e *Executor) handleDnsProbe(ctx context.Context, task *Task) (result TaskResult) {
	timeout := task.ProbeConfig.Timeout
	if timeout <= 0 {
		timeout = 5 // 默认5秒
	}

	e.logger.Info("Handling DNS probe", zap.String("task", task.ID),
		zap.String("target", task.ProbeConfig.Target), zap.Int("timeout", int(timeout)))

	// 创建带超时的上下文
	ctxWithTimeout, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()

	// 调用外部DNS探测服务
	req := &dns.Request{
		Target:  task.ProbeConfig.Target,
		Timeout: timeout,
	}

	e.logger.Info("Sending DnsProbe request", zap.Any("request", req))

	resp, err := e.tools.DNS.Probe(ctxWithTimeout, req)

	if err != nil {
		result.Error = err.Error()
		e.logger.Error("DnsProbe failed", zap.String("task", task.ID), zap.Error(err))
		return
	}

	result.Success = resp.Success
	result.DnsIps = resp.Ips
	result.Duration = resp.Duration
	result.Type = "dns"

	// 添加成功日志
	e.logger.Info("DnsProbe succeeded", zap.String("task", task.ID), zap.String("target", task.ProbeConfig.Target),
		zap.Bool("status", resp.Success))
	return result
}

func (e *Executor) handlePingProbe(ctx context.Context, task *Task) (result TaskResult) {
	timeout := task.ProbeConfig.Timeout
	if timeout <= 0 {
		timeout = 5 // 默认5秒
	}

	e.logger.Info("Handling PING probe", zap.String("task", task.ID),
		zap.String("target", task.ProbeConfig.Target), zap.Int("timeout", int(timeout)))

	// 创建带超时的上下文
	ctxWithTimeout, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()

	// 调用外部PING探测服务
	req := &ping.Request{
		Target:  task.ProbeConfig.Target,
		Timeout: timeout,
	}

	e.logger.Info("Sending PingProbe request", zap.Any("request", req))

	resp, err := e.tools.PING.Probe(ctxWithTimeout, req)

	if err != nil {
		result.Error = err.Error()
		e.logger.Error("PingProbe failed", zap.String("task", task.ID), zap.Error(err))
		return
	}

	result.Success = resp.Success
	result.Duration = resp.Duration
	result.Type = "ping"

	// 添加成功日志
	e.logger.Info("PingProbe succeeded", zap.String("task", task.ID), zap.String("target", task.ProbeConfig.Target),
		zap.Bool("status", resp.Success))
	return result
}
