package model

type User struct {
	ID    string `gorm:"primaryKey" json:"id"`
	Name  string `gorm:"size:100" json:"name"`
	Email string `gorm:"size:191;uniqueIndex;not null" json:"email"` // 设置长度限制
	MD5   string `gorm:"size:32;comment:'邮件md5两次'" json:"md5"`       // 设置为 varchar(32)
}
