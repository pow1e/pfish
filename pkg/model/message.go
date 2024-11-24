package model

import "time"

type Message struct {
	Uid         string    `json:"uid"`                   // uuid
	Computer    string    `json:"content"`               // 用户名
	Picture     string    `json:"picture_path"`          // 点击截图
	PID         string    `json:"pid" gorm:"column:pid"` // pid
	ProcessName string    `json:"process_name"`          // 进程名
	Internal    string    `json:"internal"`              // ip
	ClickTime   time.Time `json:"click_time(点击时间)"`      // 点击时间
}
