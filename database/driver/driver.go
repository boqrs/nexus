package driver

import (
	config "github.com/boqrs/nexus/config/v2"
	"database/sql"
	"errors"
	"gorm.io/gorm"
)

type GormDriver interface {
	Init(cfg config.DBConfig) gorm.Dialector
	GetSQLDB(db *gorm.DB) (*sql.DB, error)
}

type BaseDriver struct{}

func (b *BaseDriver) GetSQLDB(db *gorm.DB) (*sql.DB, error) {
	sqlDB, err := db.DB()
	if err != nil {
		return nil, errors.New("get underlying sql.DB failed")
	}
	return sqlDB, nil
}
