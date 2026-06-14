// Package email 邮件发送（MVP：未配置 SMTP 时打印到日志）。
package email

import (
	"fmt"
	"log"
	"net/smtp"
)

// Config SMTP 配置。
type Config struct {
	Enabled  bool
	Host     string
	Port     int
	Username string
	Password string
	From     string
}

// Sender 邮件发送器。
type Sender struct {
	cfg Config
}

// NewSender 创建邮件发送器。
func NewSender(cfg Config) *Sender {
	return &Sender{cfg: cfg}
}

// SendVerificationCode 发送邮箱验证码。
func (s *Sender) SendVerificationCode(to, code string) error {
	subject := "CopyFlow 验证码"
	body := fmt.Sprintf("您的 CopyFlow 验证码是：%s，5 分钟内有效。", code)
	if !s.cfg.Enabled {
		log.Printf("[email][dev] to=%s code=%s", to, code)
		return nil
	}
	msg := []byte(fmt.Sprintf("To: %s\r\nSubject: %s\r\n\r\n%s", to, subject, body))
	addr := fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port)
	auth := smtp.PlainAuth("", s.cfg.Username, s.cfg.Password, s.cfg.Host)
	return smtp.SendMail(addr, auth, s.cfg.From, []string{to}, msg)
}
