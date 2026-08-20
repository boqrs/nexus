package media

import (
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/mitchellh/mapstructure"
)

// Provider 提供可热替换的存储客户端。
//
// 业务层通过 Provider.Get() 获取当前最新的 Storage 实例。
// Provider 实现了 comm/config.ConfigReloader，可以在配置变更时自动更新底层客户端。
type Provider struct {
	key string
	val atomic.Value // Storage
	mu  sync.Mutex
}

type storageHolder struct {
	storage Storage
}

// NewProvider 创建一个 Storage Provider。
//
// 默认使用配置 key "media_cfg" 进行热更新。
func NewProvider(cfg *Config) (*Provider, error) {
	return NewProviderWithKey("media_cfg", cfg)
}

// NewProviderWithKey 创建一个 Storage Provider，并指定热更新时读取的配置 key。
func NewProviderWithKey(key string, cfg *Config) (*Provider, error) {
	if cfg == nil {
		return nil, fmt.Errorf("media config is nil")
	}

	st, err := NewStorage(cfg)
	if err != nil {
		return nil, err
	}

	p := &Provider{key: key}
	p.val.Store(storageHolder{storage: st})
	return p, nil
}

// Get 返回当前的 Storage 实例。
func (p *Provider) Get() Storage {
	v := p.val.Load()
	if v == nil {
		return nil
	}
	return v.(storageHolder).storage
}

// Update 用新配置重新创建 Storage。
func (p *Provider) Update(cfg *Config) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	st, err := NewStorage(cfg)
	if err != nil {
		return err
	}

	p.val.Store(storageHolder{storage: st})
	return nil
}

// Reload 实现 comm/config.ConfigReloader。
//
// 如果配置包含 provider.Key 对应的字段（默认是 "media_cfg"），则会从该字段解析。
func (p *Provider) Reload(cfg map[string]interface{}) error {
	subCfg := cfg
	if p.key != "" {
		if raw, ok := cfg[p.key]; ok {
			if m, ok := raw.(map[string]interface{}); ok {
				subCfg = m
			}
		}
	}

	var c Config
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Result:           &c,
		WeaklyTypedInput: true,
		TagName:          "mapstructure",
	})
	if err != nil {
		return err
	}

	if err := decoder.Decode(subCfg); err != nil {
		return err
	}

	return p.Update(&c)
}
