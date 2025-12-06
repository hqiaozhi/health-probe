package task

import "health-probe/internal/svc"

type Task struct {
	svcCtx *svc.ServiceContext
}

func New(svcCtx *svc.ServiceContext) *Task {
	return &Task{svcCtx: svcCtx}
}
