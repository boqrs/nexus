package configv2

import (
	"gorm.io/gorm/logger"
	"time"
)

type DBConfig struct {
	Driver      string            `json:"driver" yaml:"driver" mapstructure:"driver"`
	DSN         string            `json:"dsn,omitempty" yaml:"dsn,omitempty" mapstructure:"dsn"`
	Host        string            `json:"host,omitempty" yaml:"host,omitempty" mapstructure:"host"`
	Port        int               `json:"port,omitempty" yaml:"port,omitempty" mapstructure:"port"`
	Username    string            `json:"username,omitempty" yaml:"username,omitempty" mapstructure:"username"`
	Password    string            `json:"password,omitempty" yaml:"password,omitempty" mapstructure:"password"`
	Database    string            `json:"database,omitempty" yaml:"database,omitempty" mapstructure:"database"`
	Charset     string            `json:"charset,omitempty" yaml:"charset,omitempty"`
	SSLMode     string            `json:"ssl_mode" yaml:"ssl_mode" mapstructure:"ssl_mode"` // 新增
	MaxOpen     int               `json:"max_open" yaml:"max_open" mapstructure:"max_open"`
	MaxIdle     int               `json:"max_idle" yaml:"max_idle" mapstructure:"max_idle"`
	ConnMaxLife time.Duration     `json:"conn_max_life" yaml:"conn_max_life" mapstructure:"conn_max_life"`
	LogLevel    logger.LogLevel   `json:"log_level,omitempty" yaml:"log_level,omitempty" mapstructure:"log_level"`
	Params      map[string]string `json:"params,omitempty" yaml:"params,omitempty" mapstructure:"params"`
}

func (c *DBConfig) SetDefault() {
	// 通用连接池默认值
	if c.MaxOpen == 0 {
		c.MaxOpen = 10
	}
	if c.MaxIdle == 0 {
		c.MaxIdle = 5
	}
	if c.ConnMaxLife == 0 {
		c.ConnMaxLife = 30 * time.Minute
	}
	// MySQL专属默认值
	if c.Driver == "mysql" && c.Charset == "" {
		c.Charset = "utf8mb4"
	}
	// 日志级别默认值
	if c.LogLevel == 0 {
		c.LogLevel = logger.Warn
	}
}
