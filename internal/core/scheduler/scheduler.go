package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
)

// 任务类型
type TaskType int32

const (
	TaskType_IMMEDIATE TaskType = 0 // 立即执行
	TaskType_CRON      TaskType = 1 // 定时任务（cron表达式）
	TaskType_RECURRING TaskType = 2 // 循环任务（固定间隔）
	TaskType_PERIODIC  TaskType = 3 // 周期定时循环（cron+起止时间）
)

// Enum value maps for TaskType.
var (
	TaskType_name = map[int32]string{
		0: "IMMEDIATE",
		1: "CRON",
		2: "RECURRING",
		3: "PERIODIC",
	}
	TaskType_value = map[string]int32{
		"IMMEDIATE": 0,
		"CRON":      1,
		"RECURRING": 2,
		"PERIODIC":  3,
	}
)

// 任务状态
type TaskStatus int32

const (
	TaskStatus_SCHEDULED TaskStatus = 0 // 已调度
	TaskStatus_PAUSED    TaskStatus = 1 // 已暂停
	TaskStatus_STOPPED   TaskStatus = 2 // 已停止
)

// Enum value maps for TaskStatus.
var (
	TaskStatus_name = map[int32]string{
		0: "SCHEDULED",
		1: "PAUSED",
		2: "STOPPED",
	}
	TaskStatus_value = map[string]int32{
		"SCHEDULED": 0,
		"PAUSED":    1,
		"STOPPED":   2,
	}
)

// 探测类型
type ProbeType int32

const (
	ProbeType_HTTP ProbeType = 0
	ProbeType_TCP  ProbeType = 1
	ProbeType_DNS  ProbeType = 2
	ProbeType_PING ProbeType = 3
)

// Enum value maps for ProbeType.
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

// 任务定义
type Task struct {
	ID             string
	Name           string
	Type           TaskType
	ProbeConfig    *ProbeConfig
	CronExpression string
	StartTime      int64
	EndTime        int64
	Status         TaskStatus
}

// 探测配置
type ProbeConfig struct {
	Type                  ProbeType
	Target                string
	Tls                   bool
	Port                  int64
	Timeout               int64
	FailureWindowDuration int64
	SuccessWindowDuration int64
	FailureThreshold      int
	AlertEmailRecipients  []string
}

// 任务结果
type TaskResult struct {
	TaskId   string
	Success  bool
	Error    string
	ExecTime int64
	Duration int64
	// 扩展字段，根据探测类型填充
	HttpStatus int32
	DnsIps     []string // DNS解析IP列表（仅DNS探测）
	Type       string   // 探测类型
}

// 任务信息结构体，包含任务配置和调度信息
type TaskInfo struct {
	Task    *Task        // 任务配置
	EntryID cron.EntryID // cron条目ID
	Paused  bool         // 是否暂停
}

// Scheduler 任务调度器
type Scheduler struct {
	cron     *cron.Cron
	executor *Executor
	taskMap  map[string]*TaskInfo // 任务ID -> 任务信息
	mu       sync.RWMutex
}

// NewScheduler 创建调度器实例
func NewScheduler(executor *Executor) IScheduler {
	return &Scheduler{
		cron:     cron.New(),
		executor: executor,
		taskMap:  make(map[string]*TaskInfo),
	}
}

// Start 启动调度器
func (s *Scheduler) Start() {
	s.cron.Start()
}

// 接口
type IScheduler interface {
	Start()
	CreateTask(ctx context.Context, task *Task) (string, error)
	DeleteTask(taskID string) error
	UpdateTask(ctx context.Context, task *Task) error
	QueryTasks(taskID string, status []TaskStatus) ([]*Task, int)
	PauseTask(taskID string) error
	ResumeTask(ctx context.Context, taskID string) error
}

// CreateTask 创建并调度任务
func (s *Scheduler) CreateTask(ctx context.Context, task *Task) (string, error) {
	// 检查任务是否已存在
	s.mu.RLock()
	_, exists := s.taskMap[task.ID]
	s.mu.RUnlock()
	if exists {
		return "", fmt.Errorf("task %s already exists", task.ID)
	}

	// 设置初始状态
	task.Status = TaskStatus_SCHEDULED

	// 创建执行函数
	// 使用背景上下文而不是传入的ctx，避免外部取消影响任务执行
	taskCtx := context.Background()
	execFunc := func() {
		s.executor.Execute(taskCtx, task)
	}

	var entryID cron.EntryID
	var err error
	var taskInfo *TaskInfo

	switch task.Type {
	case TaskType_IMMEDIATE:
		// 立即执行任务
		go execFunc()
		// 立即任务执行后状态设为已完成
		task.Status = TaskStatus_STOPPED
		taskInfo = &TaskInfo{
			Task:   task,
			Paused: false,
		}

	case TaskType_CRON:
		// 定时任务
		if task.CronExpression == "" {
			return "", fmt.Errorf("cron task requires cron_expression")
		}
		entryID, err = s.cron.AddFunc(task.CronExpression, execFunc)
		taskInfo = &TaskInfo{
			Task:    task,
			EntryID: entryID,
			Paused:  false,
		}

	case TaskType_RECURRING:
		entryID, err = s.cron.AddFunc(task.CronExpression, execFunc)
		taskInfo = &TaskInfo{
			Task:    task,
			EntryID: entryID,
			Paused:  false,
		}

	case TaskType_PERIODIC:
		// 周期定时循环任务
		if task.CronExpression == "" || task.StartTime <= 0 || task.EndTime <= 0 || task.StartTime >= task.EndTime {
			return "", fmt.Errorf("invalid periodic task parameters")
		}
		// 包装执行函数，增加周期判断
		wrappedFunc := func() {
			now := time.Now().Unix()
			if now >= task.StartTime && now <= task.EndTime {
				execFunc()
			} else if now > task.EndTime {
				// 超出周期自动停止
				s.DeleteTask(task.ID)
			}
		}
		entryID, err = s.cron.AddFunc(task.CronExpression, wrappedFunc)
		taskInfo = &TaskInfo{
			Task:    task,
			EntryID: entryID,
			Paused:  false,
		}

	default:
		return "", fmt.Errorf("unsupported task type: %v", task.Type)
	}

	if err != nil {
		return "", fmt.Errorf("failed to schedule task: %v", err)
	}

	// 记录任务
	s.mu.Lock()
	s.taskMap[task.ID] = taskInfo
	s.mu.Unlock()

	return task.ID, nil
}

// DeleteTask 停止并删除任务
func (s *Scheduler) DeleteTask(taskID string) error {
	s.mu.RLock()
	taskInfo, exists := s.taskMap[taskID]
	s.mu.RUnlock()

	if !exists {
		return fmt.Errorf("task %s not found", taskID)
	}

	// 从cron中移除（如果不是立即执行任务）
	if taskInfo.Task.Type != TaskType_IMMEDIATE && !taskInfo.Paused {
		s.cron.Remove(taskInfo.EntryID)
	}

	// 从映射中删除
	s.mu.Lock()
	delete(s.taskMap, taskID)
	s.mu.Unlock()

	return nil
}

// UpdateTask 更新任务配置
func (s *Scheduler) UpdateTask(ctx context.Context, task *Task) error {
	if task.ID == "" {
		return fmt.Errorf("task id is required")
	}

	// 先删除旧任务（如果存在）
	deleteErr := s.DeleteTask(task.ID)
	// 如果删除失败且错误不是"任务不存在"，则返回错误
	if deleteErr != nil && deleteErr.Error() != fmt.Sprintf("task %s not found", task.ID) {
		return deleteErr
	}

	// 再创建新任务
	_, err := s.CreateTask(ctx, task)
	return err
}

// QueryTasks 查询任务
func (s *Scheduler) QueryTasks(taskID string, status []TaskStatus) ([]*Task, int) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var tasks []*Task

	// 筛选状态
	statusMap := make(map[TaskStatus]bool)
	for _, s := range status {
		statusMap[s] = true
	}

	// 如果指定了任务ID，优先查询单个任务
	if taskID != "" {
		if taskInfo, exists := s.taskMap[taskID]; exists {
			// 检查状态是否匹配
			if len(status) == 0 || statusMap[taskInfo.Task.Status] {
				tasks = append(tasks, taskInfo.Task)
			}
		}
		return tasks, len(tasks)
	}

	// 查询所有任务
	for _, taskInfo := range s.taskMap {
		// 检查状态是否匹配
		if len(status) == 0 || statusMap[taskInfo.Task.Status] {
			tasks = append(tasks, taskInfo.Task)
		}
	}

	return tasks, len(tasks)
}

// PauseTask 暂停任务
func (s *Scheduler) PauseTask(taskID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	taskInfo, exists := s.taskMap[taskID]
	if !exists {
		return fmt.Errorf("task %s not found", taskID)
	}

	if taskInfo.Paused {
		return fmt.Errorf("task %s is already paused", taskID)
	}

	// 立即执行任务不能暂停
	if taskInfo.Task.Type == TaskType_IMMEDIATE {
		return fmt.Errorf("cannot pause immediate tasks")
	}

	// 从cron中移除（停止执行）
	s.cron.Remove(taskInfo.EntryID)
	taskInfo.Paused = true
	taskInfo.Task.Status = TaskStatus_PAUSED
	s.taskMap[taskID] = taskInfo

	return nil
}

// ResumeTask 恢复任务执行
func (s *Scheduler) ResumeTask(ctx context.Context, taskID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	taskInfo, exists := s.taskMap[taskID]
	if !exists {
		return fmt.Errorf("task %s not found", taskID)
	}

	if !taskInfo.Paused {
		return fmt.Errorf("task %s is not paused", taskID)
	}

	// 创建执行函数
	execFunc := func() {
		s.executor.Execute(ctx, taskInfo.Task)
	}

	var entryID cron.EntryID
	var err error

	// 根据任务类型重新添加到cron
	switch taskInfo.Task.Type {
	case TaskType_CRON:
		entryID, err = s.cron.AddFunc(taskInfo.Task.CronExpression, execFunc)

	case TaskType_RECURRING:
		entryID, err = s.cron.AddFunc(taskInfo.Task.CronExpression, execFunc)

	case TaskType_PERIODIC:
		wrappedFunc := func() {
			now := time.Now().Unix()
			if now >= taskInfo.Task.StartTime && now <= taskInfo.Task.EndTime {
				execFunc()
			} else if now > taskInfo.Task.EndTime {
				s.DeleteTask(taskID)
			}
		}
		entryID, err = s.cron.AddFunc(taskInfo.Task.CronExpression, wrappedFunc)

	default:
		return fmt.Errorf("cannot resume task type: %v", taskInfo.Task.Type)
	}

	if err != nil {
		return fmt.Errorf("failed to resume task: %v", err)
	}

	// 更新任务信息（恢复运行）
	taskInfo.EntryID = entryID
	taskInfo.Paused = false
	taskInfo.Task.Status = TaskStatus_SCHEDULED
	s.taskMap[taskID] = taskInfo

	return nil
}
