package models

import "gorm.io/gorm"

// DAO 统一DAO入口（聚合所有独立DAO）
type DAO struct {
	Task     TaskDAO     // 任务DAO
	EmailLog EmailLogDAO // 邮件日志DAO
	// 后续新增其他DAO（如用户DAO、配置DAO），直接在此添加字段即可
}

// NewDAO 创建统一DAO实例（一次性初始化所有独立DAO）
func NewDAO(db *gorm.DB) *DAO {
	return &DAO{
		Task:     NewTaskDAO(db),     // 初始化任务DAO
		EmailLog: NewEmailLogDAO(db), // 初始化邮件日志DAO
	}
}
