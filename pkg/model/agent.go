package model

import "time"

type AgentConfig struct {
	TemplateID   string    `json:"template_id" gorm:"primaryKey"`
	OpenFileName string    `json:"open_file_name" gorm:"size:255;not null"`
	Content      []byte    `json:"content" gorm:"type:longblob"` // 文件内容
	UseTaskID    string    `json:"use_task_id"`                  // 使用的任务id
	CreatedAt    time.Time `gorm:"autoCreateTime"`               // 创建时间
}
