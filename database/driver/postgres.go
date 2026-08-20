package driver

import (
	"fmt"

	config "codeup.aliyun.com/65b21d33076e069afe3d3253/basice/comm/config/v2"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type PostgresDriver struct {
	BaseDriver
}

func (p *PostgresDriver) Init(cfg config.DBConfig) gorm.Dialector {
	if cfg.DSN != "" {
		return postgres.Open(cfg.DSN)
	}

	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable TimeZone=Asia/Shanghai",
		cfg.Host, cfg.Port, cfg.Username, cfg.Password, cfg.Database,
	)
	for k, v := range cfg.Params {
		dsn += fmt.Sprintf(" %s=%s", k, v)
	}
	return postgres.Open(dsn)
}
