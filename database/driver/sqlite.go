package driver

import (
	config "codeup.aliyun.com/65b21d33076e069afe3d3253/basice/comm/config/v2"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type SQLiteDriver struct {
	BaseDriver
}

func (s *SQLiteDriver) Init(cfg config.DBConfig) gorm.Dialector {
	if cfg.DSN != "" {
		return sqlite.Open(cfg.DSN)
	}
	return sqlite.Open(cfg.Database)
}
