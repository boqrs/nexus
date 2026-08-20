package database

import (
	"context"
	"fmt"
	"time"

	config "codeup.aliyun.com/65b21d33076e069afe3d3253/basice/comm/config/v2"
	"codeup.aliyun.com/65b21d33076e069afe3d3253/basice/comm/database/driver"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func NewGormDB(cfg config.DBConfig) (*gorm.DB, error) {
	cfg.SetDefault()

	var d driver.GormDriver
	switch cfg.Driver {
	case "mysql":
		d = &driver.MySQLDriver{}
	case "postgres":
		d = &driver.PostgresDriver{}
	case "sqlite":
		d = &driver.SQLiteDriver{}
	default:
		return nil, fmt.Errorf("unsupported driver: %s", cfg.Driver)
	}

	dialector := d.Init(cfg)

	gormDB, err := gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(cfg.LogLevel),
	})
	if err != nil {
		return nil, fmt.Errorf("init sql driver failed: %w", err)
	}

	sqlDB, err := d.GetSQLDB(gormDB)
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(cfg.MaxOpen)
	sqlDB.SetMaxIdleConns(cfg.MaxIdle)
	sqlDB.SetConnMaxLifetime(cfg.ConnMaxLife)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel() // 必须调用cancel，避免context泄漏
	if err := sqlDB.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("db ping failed: %w", err)
	}

	return gormDB, nil
}
