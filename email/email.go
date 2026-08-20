package email

import (
	"github.com/wneessen/go-mail"
)

type Config struct {
	Server   string `json:"server"`
	Username string `json:"username"`
	Password string `json:"password"`
}

func Init(cfg Config) (*mail.Client, error) {
	return mail.NewClient(cfg.Server,
		mail.WithSSLPort(true),
		mail.WithSMTPAuth(mail.SMTPAuthPlain),
		mail.WithUsername(cfg.Username),
		mail.WithPassword(cfg.Password))
}
