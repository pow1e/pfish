package model

import (
	"gorm.io/gorm"
	"time"
)

type Task struct {
	gorm.Model
	Name      string    `json:"name"`       // 任务名称
	StartTime time.Time `json:"start_time"` // 运行时间
	Status    bool      `json:"status"`     // 是否启动
}
