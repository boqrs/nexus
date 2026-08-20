package redis

import (
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
)

// Provider 提供可热替换的 Redis 客户端实例。
//
// 业务层通过 Provider.Get() 获取当前最新的 redis.Client。
// Provider 实现了 comm/config.ConfigReloader，可以在配置变更时自动更新底层客户端。
//
// 该实现会在更新时关闭旧的客户端。
type Provider struct {
	key string
	val atomic.Value // Client
	mu  sync.Mutex
}

// NewProvider 创建一个 Provider 实例。
//
// 默认使用配置 key "redis_cfg" 进行热更新。
func NewProvider(cfg *Config) (*Provider, error) {
	return NewProviderWithKey("redis_cfg", cfg)
}

// NewProviderWithKey 创建一个 Provider 实例，并指定热更新时读取配置的 key。
func NewProviderWithKey(key string, cfg *Config) (*Provider, error) {
	if cfg == nil {
		return nil, fmt.Errorf("redis config is nil")
	}

	cli, err := NewClient(cfg)
	if err != nil {
		return nil, err
	}

	p := &Provider{key: key}
	p.val.Store(cli)
	return p, nil
}

// Get 返回当前的 Redis 客户端。
func (p *Provider) Get() Client {
	v := p.val.Load()
	if v == nil {
		return nil
	}
	return v.(Client)
}

// Update 用新配置重新创建客户端，并关闭旧客户端。
func (p *Provider) Update(cfg *Config) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	cli, err := NewClient(cfg)
	if err != nil {
		return err
	}

	old := p.Get()
	p.val.Store(cli)

	if old != nil {
		_ = old.Close()
	}

	return nil
}

// Reload 实现 comm/config.ConfigReloader。
//
// 如果配置包含 provider.Key 对应的字段（默认是 "redis_cfg"），则会使用该字段进行解析。
// 否则会尝试直接解析传入的 map。
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

	var c Config
	if err := json.Unmarshal(b, &c); err != nil {
		return err
	}

	return p.Update(&c)
}

// Close 关闭当前客户端。
func (p *Provider) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	old := p.Get()
	if old == nil {
		return nil
	}
	return old.Close()
}
