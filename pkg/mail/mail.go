package mail

import (
	"github.com/pow1e/pfish/config"
	"gopkg.in/gomail.v2"
)

type Mailer struct {
	From     string
	To       string
	Subject  string
	Template string
	Data     interface{}
}

func sendMail(to string) error {
	// 创建邮件
	m := gomail.NewMessage()
	// 设置发送者
	m.SetHeader("From", config.Conf.SMTP.User)
	// 设置接收者
	m.SetHeader("To", to)
	// 设置邮件主题
	m.SetHeader("Subject", "测试邮件")
	// 邮件正文
	m.SetBody("text/plain", "这是一封通过Go语言发送的测试邮件。")
	m.SetHeader("X-Mailer", "CustomMailer v1.0")

	// 附件 (可选)
	// m.Attach("/path/to/file")

	// 发送邮件
	smtpConf := config.Conf.SMTP
	d := gomail.NewDialer(smtpConf.Host, smtpConf.Port, smtpConf.User, smtpConf.Password)
	if err := d.DialAndSend(m); err != nil {
		return err
	}
	println("邮件发送成功!")
	return nil
}
