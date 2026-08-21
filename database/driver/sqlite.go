package driver

import (
	config "github.com/boqrs/nexus/config/v2"
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
