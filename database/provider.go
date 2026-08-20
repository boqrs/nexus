package database

import (
	"encoding/json"
	"sync"
	"sync/atomic"

	config "codeup.aliyun.com/65b21d33076e069afe3d3253/basice/comm/config/v2"
	"gorm.io/gorm"
)

// DBProvider provides a reloadable *gorm.DB instance.
//
// It is intended to be passed around into services/handlers; callers always
// call Get() to acquire the current active DB instance.
//
// It also implements comm/config.ConfigReloader so it can be used with
// config manager hot-reload logic.
//
// Note: Reload expects a map that represents the DBConfig fields.
// For example, the map might be the value of the "db_cfg" key in your app config.
//
// When updated, the old *gorm.DB is closed (if possible).
//
// Usage:
//
//	dbProvider, _ := database.NewProvider(cfg)
//	db := dbProvider.Get()
//	// ...
//	configManager.AddReloader(dbProvider)
//
// With config watcher, Reload() will be called with the latest db config map.
type DBProvider struct {
	key string
	val atomic.Value // *gorm.DB
	mu  sync.Mutex
}

// NewProvider creates a new DBProvider initialized with cfg.
//
// It sets the default config key to "db_cfg" when reloading.
func NewProvider(cfg config.DBConfig) (*DBProvider, error) {
	return NewProviderWithKey("db_cfg", cfg)
}

// NewProviderWithKey creates a new DBProvider that expects the db config
// to be under the given key in the global configuration map.
func NewProviderWithKey(key string, cfg config.DBConfig) (*DBProvider, error) {
	cfg.SetDefault()
	db, err := NewGormDB(cfg)
	if err != nil {
		return nil, err
	}
	p := &DBProvider{key: key}
	p.val.Store(db)
	return p, nil
}

// Get returns the current *gorm.DB instance.
func (p *DBProvider) Get() *gorm.DB {
	v := p.val.Load()
	if v == nil {
		return nil
	}
	return v.(*gorm.DB)
}

// Update replaces the current DB instance with a new one created from cfg.
// It closes the previous DB connection if possible.
func (p *DBProvider) Update(cfg config.DBConfig) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	cfg.SetDefault()
	newDB, err := NewGormDB(cfg)
	if err != nil {
		return err
	}

	old := p.Get()
	p.val.Store(newDB)

	if old != nil {
		if sqlDB, err := old.DB(); err == nil {
			_ = sqlDB.Close()
		}
	}

	return nil
}

// Reload implements comm/config.ConfigReloader.
//
// If the passed map contains the configured key (default: "db_cfg"), it will
// extract that sub-map before unmarshalling. Otherwise, it will attempt to
// unmarshal the top-level map directly.
func (p *DBProvider) Reload(cfg map[string]interface{}) error {
	subCfg := cfg
	if p.key != "" {
		if raw, ok := cfg[p.key]; ok {
			if m, ok := raw.(map[string]interface{}); ok {
				subCfg = m
			}
		}
	}

	b, err := json.Marshal(subCfg)
	if err != nil {
		return err
	}

	var c config.DBConfig
	if err := json.Unmarshal(b, &c); err != nil {
		return err
	}

	return p.Update(c)
}

// Close closes the current DB instance if it exists.
func (p *DBProvider) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	old := p.Get()
	if old == nil {
		return nil
	}
	if sqlDB, err := old.DB(); err == nil {
		return sqlDB.Close()
	}
	return nil
}
