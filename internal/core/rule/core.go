package rule

import (
	"context"
	"fmt"
	"health-probe/internal/core/notify/email"
	"health-probe/internal/models"
	"health-probe/internal/utils/alert"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
	lru "github.com/hashicorp/golang-lru/v2"
	"go.uber.org/zap"
)

// 告警结果：推送给外部的通知数据
type AlertResult struct {
	TaskID            string // 任务ID
	NeedFailureAlert  bool   // 是否需要发送失败告警
	NeedSuccessNotify bool   // 是否需要发送成功恢复通知
	ErrMsg            string // 错误信息（参数校验失败等场景）
}

// 探测参数：外部调用 Probe 时传入的参数
type ProbeParam struct {
	TaskID                string // 任务唯一标识
	TaskName              string // 任务名称
	Target                string // 目标URL（HTTP/HTTPS/GRPC）
	ProbeType             models.ProbeType
	CurrentTimeMs         int64    // 当前探测时间（毫秒级时间戳）
	IsSuccess             bool     // 探测是否成功
	FailureWindowDuration int64    // 失败统计窗口时长（毫秒）
	SuccessWindowDuration int64    // 成功恢复窗口时长（毫秒）
	FailureThreshold      int      // 失败告警阈值（窗口内失败次数达到该值触发告警）
	AlertEmailRecipients  []string // 告警接收人邮箱列表
}

// 告警统计服务接口（定义核心能力）
type AlertStatService interface {
	Probe(param ProbeParam) error // 接收探测结果
}

// 循环队列：滑动窗口的核心存储容器（存储失败时间戳）
type CircularQueue struct {
	buffer []int64 // 存储失败时间戳（毫秒）
	cap    int     // 队列最大容量（= 失败阈值+1）
	head   int     // 队头指针（出队方向）
	tail   int     // 队尾指针（入队方向）
	count  int     // 当前队列元素个数
}

// 任务状态：每个任务的滑动窗口数据+告警状态
type TaskState struct {
	mu                  sync.RWMutex   // 保护当前任务状态的并发访问
	failureTimestamps   *CircularQueue // 失败时间戳队列（滑动窗口核心）
	failureAlertStatus  bool           // 失败告警是否已发送（避免重复告警）
	successNotifyStatus bool           // 成功恢复通知是否已发送（避免重复通知）
	lastProbeTimeMs     int64          // 最近一次探测时间（毫秒）
	hasValidProbe       bool           // 是否有过有效探测
	hasFailureRecord    bool           // 是否有过失败记录
	firstSuccessTimeMs  int64          // 失败后首次成功的时间戳（毫秒）
	isProtected         bool           // 核心任务标记（保护任务不被LRU淘汰）
}

// -------------------------- 循环队列实现（原有逻辑不变） --------------------------
// newCircularQueue 创建指定容量的循环队列
func newCircularQueue(cap int) *CircularQueue {
	if cap <= 0 {
		cap = 10 // 默认容量兜底
	}
	return &CircularQueue{
		buffer: make([]int64, cap),
		cap:    cap,
		head:   0,
		tail:   0,
		count:  0,
	}
}

// enqueue 入队：添加时间戳到队尾（队列满时覆盖队头旧数据）
func (q *CircularQueue) enqueue(timestamp int64) {
	q.buffer[q.tail] = timestamp
	q.tail = (q.tail + 1) % q.cap // 循环移动队尾指针

	if q.count == q.cap {
		q.head = (q.head + 1) % q.cap // 队列满，队头后移（覆盖最旧元素）
	} else {
		q.count++
	}
}

// removeExpired 清理过期元素：删除早于 expireTimeMs 的时间戳，返回当前有效元素数
func (q *CircularQueue) removeExpired(expireTimeMs int64) int {
	if q.count == 0 {
		return 0
	}

	// 循环队列按时间递增存储，找到第一个未过期的元素（后续均未过期）
	expiredCount := 0
	for i := 0; i < q.count; i++ {
		idx := (q.head + i) % q.cap
		if q.buffer[idx] < expireTimeMs {
			expiredCount++
		} else {
			break
		}
	}

	// 移动队头指针，跳过过期元素
	q.head = (q.head + expiredCount) % q.cap
	q.count -= expiredCount

	return q.count
}

// size 返回当前队列有效元素个数
func (q *CircularQueue) size() int {
	return q.count
}

// -------------------------- LRU集成后的内存版服务实现 --------------------------
// MemoryAlertStatService 内存版告警统计服务（集成LRU自动清理）
type MemoryAlertStatService struct {
	mu           sync.RWMutex                   // 保护LRU缓存的并发访问（兼容原有锁逻辑）
	taskStates   *lru.Cache[string, *TaskState] // LRU缓存（key: TaskID）
	maxCacheSize int                            // LRU最大容量（控制最大任务数）
	email        email.Sender                   // 告警通知发送器
	ctx          context.Context                // 上下文（用于取消操作）
	DAO          *models.DAO
	logger       *zap.Logger
}

// NewMemoryAlertStatService 创建LRU版告警统计服务
// maxCacheSize: LRU最大容量（建议按 单任务内存占用 × 容量 ≤ 可用内存 配置，如10000）
func NewMemoryAlertStatService(DAO *models.DAO, email email.Sender, maxCacheSize int, logger *zap.Logger) (*MemoryAlertStatService, error) {

	if maxCacheSize <= 0 {
		return nil, fmt.Errorf("maxCacheSize must be positive")
	}

	// 先创建service实例
	service := &MemoryAlertStatService{
		maxCacheSize: maxCacheSize,
		email:        email,
		ctx:          context.Background(),
		DAO:          DAO,
		logger:       logger,
	}

	// 初始化LRU缓存，添加淘汰回调（支持保护核心任务）
	lruCache, err := lru.NewWithEvict(
		maxCacheSize,
		func(taskID string, state *TaskState) {
			// 保护任务：拒绝淘汰，重新放入缓存
			if state.isProtected {
				service.mu.Lock()
				service.taskStates.Add(taskID, state)
				service.mu.Unlock()
				service.logger.Info("[LRU] 任务[%s]为核心保护任务，拒绝淘汰", zap.String("taskID", taskID))
				return
			}
			service.logger.Info("[LRU] 自动淘汰任务[%s]（最久未使用）", zap.String("taskID", taskID))
		},
	)
	if err != nil {
		return nil, fmt.Errorf("initialize LRU cache failed: %w", err)
	}

	service.taskStates = lruCache

	return service, nil
}

// Probe 接收探测结果（核心入口）
func (m *MemoryAlertStatService) Probe(param ProbeParam) error {
	// 1. 入参校验
	if err := m.validateParam(param); err != nil {
		return err
	}

	// 2. 异步处理（避免阻塞外部）
	go m.processProbe(param)

	return nil
}

// validateParam 入参校验（独立方法，便于维护）
func (m *MemoryAlertStatService) validateParam(param ProbeParam) error {
	if param.TaskID == "" {
		return fmt.Errorf("taskID cannot be empty")
	}
	if param.CurrentTimeMs <= 0 {
		return fmt.Errorf("currentTimeMs must be positive")
	}
	if param.FailureWindowDuration <= 0 {
		return fmt.Errorf("failureWindowDuration must be positive")
	}
	if param.SuccessWindowDuration <= 0 {
		return fmt.Errorf("successWindowDuration must be positive")
	}
	if param.FailureThreshold <= 0 {
		return fmt.Errorf("failureThreshold must be positive")
	}
	return nil
}

// processProbe 异步处理探测结果（核心逻辑，原有告警逻辑不变）
func (m *MemoryAlertStatService) processProbe(param ProbeParam) {
	// 生成顺序唯一ID
	uuid, err := uuid.NewV7()
	if err != nil {
		m.logger.Error("create uuid failed", zap.Error(err))
		os.Exit(1)
	}

	// 成功窗口默认使用失败窗口时长（兼容参数默认值）
	successWindowDuration := param.SuccessWindowDuration
	if successWindowDuration == 0 {
		successWindowDuration = param.FailureWindowDuration
	}

	// 1. 从LRU获取任务状态（Get操作自动更新"最近使用时间"，避免被淘汰）
	m.mu.RLock()
	taskState, exists := m.taskStates.Get(param.TaskID)
	m.mu.RUnlock()

	// 2. 任务不存在：创建新状态并放入LRU（缓存满时自动淘汰最久未用任务）
	if !exists {
		taskState = &TaskState{
			failureTimestamps:   newCircularQueue(param.FailureThreshold + 1),
			failureAlertStatus:  false,
			successNotifyStatus: false,
			lastProbeTimeMs:     param.CurrentTimeMs,
			hasValidProbe:       false,
			hasFailureRecord:    false,
			firstSuccessTimeMs:  0,
			isProtected:         false, // 默认非保护任务（可通过外部扩展修改）
		}
		// 放入LRU缓存
		m.mu.Lock()
		m.taskStates.Add(param.TaskID, taskState)
		m.mu.Unlock()
	}

	// 3. 加任务锁，保护状态修改（原有逻辑不变）
	taskState.mu.Lock()
	defer taskState.mu.Unlock()

	// 更新基础状态
	taskState.lastProbeTimeMs = param.CurrentTimeMs
	taskState.hasValidProbe = true
	result := AlertResult{TaskID: param.TaskID, ErrMsg: ""}

	// 4. 探测失败场景处理
	if !param.IsSuccess {
		taskState.hasFailureRecord = true
		taskState.firstSuccessTimeMs = 0 // 失败时重置成功计时

		// 清理过期失败记录 → 添加当前失败时间戳
		expireTimeMs := param.CurrentTimeMs - param.FailureWindowDuration
		taskState.failureTimestamps.removeExpired(expireTimeMs)
		taskState.failureTimestamps.enqueue(param.CurrentTimeMs)
		failureCount := taskState.failureTimestamps.size()

		// 失败告警（未发送过且达到阈值）
		if failureCount >= param.FailureThreshold && !taskState.failureAlertStatus {
			result.NeedFailureAlert = true
			taskState.failureAlertStatus = true

			failHTML, err := alert.GenerateContinuousFailAlertHTML(
				param.TaskName,
				alert.ProbeType(param.ProbeType),
				param.Target,
				param.TaskID,
				int(param.FailureWindowDuration/60000), // 转换为分钟
				failureCount,
				time.Now().Format("2006-01-02 15:04:05"),
			)
			if err != nil {
				m.logger.Error("generate continuous fail alert html failed", zap.Error(err), zap.String("taskID", param.TaskID))
				return
			}
			// 发送邮件通知（只在满足条件时发送一次）
			m.email.Send(m.ctx, &email.EmailSendRequest{
				To:      param.AlertEmailRecipients,
				Subject: "任务失败通知",
				Body:    failHTML,
				IsHTML:  true,
				Charset: "utf-8",
			})

			// 告警入库
			err = m.DAO.EmailLog.Create(m.ctx, &models.EmailLog{
				ID:         uuid.String(),
				TaskID:     param.TaskID,
				TaskName:   param.TaskName,
				ProbeType:  param.ProbeType,
				Target:     param.Target,
				AlertType:  0,
				SendStatus: 0, // 统一记录发送成功
				Recipients: param.AlertEmailRecipients,
				Message:    fmt.Sprintf(" %d 分钟内已连续失败 %d 次", param.FailureWindowDuration/60000, failureCount),
				Is_Read:    false,
			})
			if err != nil {
				m.logger.Error("create email log failed", zap.Error(err), zap.String("taskID", param.TaskID))
			} else {
				m.logger.Info("send email log success", zap.String("taskID", param.TaskID),
					zap.String("target", param.Target), zap.Int("failureCount", failureCount))
			}
		}

		return
	}

	// 5. 探测成功场景处理
	// 5.1 清理过期失败记录
	expireTimeMs := param.CurrentTimeMs - param.FailureWindowDuration
	currentFailureCount := taskState.failureTimestamps.removeExpired(expireTimeMs)

	// 5.2 初始成功（无失败记录）：重置告警状态
	if !taskState.hasFailureRecord {
		taskState.failureAlertStatus = false
		return
	}

	// 5.3 失败后首次成功：初始化计时
	if taskState.firstSuccessTimeMs == 0 {
		taskState.firstSuccessTimeMs = param.CurrentTimeMs
		taskState.failureAlertStatus = false
		taskState.successNotifyStatus = false
		m.logger.Info("任务[%s]失败后首次成功：计时起点=%d，当前窗口剩余失败=%d次",
			zap.String("taskID", param.TaskID),
			zap.Int64("firstSuccessTimeMs", taskState.firstSuccessTimeMs),
			zap.Int("currentFailureCount", currentFailureCount))
		return
	}

	// 5.4 非首次成功：计算持续成功时长
	successDurationMs := param.CurrentTimeMs - taskState.firstSuccessTimeMs

	// 5.5 触发成功恢复通知（满足所有条件）
	needSuccessNotify := !taskState.successNotifyStatus &&
		taskState.hasValidProbe &&
		successDurationMs >= successWindowDuration &&
		taskState.hasFailureRecord

	if needSuccessNotify {
		result.NeedSuccessNotify = true
		taskState.successNotifyStatus = true
		successHTML, err := alert.GenerateRecoveryAlertHTML(
			param.TaskName,
			alert.ProbeType(param.ProbeType),
			param.Target,
			param.TaskID,
			int(successDurationMs/60000), // 转换为分钟
			time.Now().Format("2006-01-02 15:04:05"),
		)
		if err != nil {
			m.logger.Error("generate success notify html failed", zap.Error(err), zap.String("taskID", param.TaskID))
			return
		}
		// 发送成功恢复通知（只在满足条件时发送一次）
		m.email.Send(m.ctx, &email.EmailSendRequest{
			To:      param.AlertEmailRecipients,
			Subject: "任务恢复通知",
			Body:    successHTML,
			IsHTML:  true,
			Charset: "utf-8",
		})
		// 告警入库
		err = m.DAO.EmailLog.Create(m.ctx, &models.EmailLog{
			ID:         uuid.String(),
			TaskID:     param.TaskID,
			TaskName:   param.TaskName,
			ProbeType:  param.ProbeType,
			Target:     param.Target,
			AlertType:  1,
			SendStatus: 0,
			Recipients: param.AlertEmailRecipients,
			Message:    fmt.Sprintf("已成功恢复，持续时间%d分钟。", successDurationMs/60000),
			Is_Read:    false,
		})
		if err != nil {
			m.logger.Error("create email log failed", zap.Error(err), zap.String("taskID", param.TaskID))
		} else {
			m.logger.Info("send email log success", zap.String("taskID", param.TaskID),
				zap.String("target", param.Target), zap.Int64("successDurationMs", successDurationMs))
		}
	}
}
