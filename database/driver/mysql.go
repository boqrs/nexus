package driver

import (
	"fmt"

	config "codeup.aliyun.com/65b21d33076e069afe3d3253/basice/comm/config/v2"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type MySQLDriver struct {
	BaseDriver
}

func (m *MySQLDriver) Init(cfg config.DBConfig) gorm.Dialector {
	if cfg.DSN != "" {
		return mysql.Open(cfg.DSN)
	}

	dsn := fmt.Sprintf(
		"%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=true&loc=Local",
		cfg.Username, cfg.Password, cfg.Host, cfg.Port,
		cfg.Database, cfg.Charset,
	)

	for k, v := range cfg.Params {
		dsn += fmt.Sprintf("&%s=%s", k, v)
	}
	return mysql.Open(dsn)
}
