package log

import (
	"encoding/json"
	"sync"
	"sync/atomic"

	zlog "github.com/boqrs/zeus/log"
)

// Provider 提供可热替换的日志实例。
//
// 业务层通过 Provider.Get() 获取当前最新的 zlog.Logger。
// Provider 实现了 comm/config.ConfigReloader，可以在配置变更时自动更新底层日志实例。
type Provider struct {
	key string
	val atomic.Value // zlog.Logger
	mu  sync.Mutex
}

// NewProvider 创建一个新的日志 Provider。
//
// 默认使用配置 key "log_cfg" 进行热更新。
func NewProvider(cfg LogConfig) (*Provider, error) {
	return NewProviderWithKey("log_cfg", cfg)
}

// NewProviderWithKey 创建一个新的日志 Provider，并指定热更新时读取的配置 key。
func NewProviderWithKey(key string, cfg LogConfig) (*Provider, error) {
	logger, err := InitLogger(cfg)
	if err != nil {
		return nil, err
	}

	p := &Provider{key: key}
	p.val.Store(logger)
	return p, nil
}

// Get 获取当前日志实例。
func (p *Provider) Get() zlog.Logger {
	v := p.val.Load()
	if v == nil {
		return nil
	}
	return v.(zlog.Logger)
}

// Update 使用新配置重新初始化日志实例。
func (p *Provider) Update(cfg LogConfig) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	logger, err := InitLogger(cfg)
	if err != nil {
		return err
	}

	p.val.Store(logger)
	return nil
}

// Reload 实现 comm/config.ConfigReloader。 
//
// 如果配置包含 provider.Key 对应的字段（默认是 "log_cfg"），则会从该字段解析。
func (p *Provider) Reload(cfg map[string]interface{}) error {
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

	var c LogConfig
	if err := json.Unmarshal(b, &c); err != nil {
		return err
	}

	return p.Update(c)
}
