package email

import (
	"encoding/json"
	"sync"
	"sync/atomic"

	"github.com/wneessen/go-mail"
)

type Provider struct {
	key string
	val atomic.Value // *mailClient
	mu  sync.Mutex
}

func NewProvider(cfg Config) (*Provider, error) {
	return NewProviderWithKey("email_cfg", cfg)
}

func NewProviderWithKey(key string, cfg Config) (*Provider, error) {
	client, err := Init(cfg)
	if err != nil {
		return nil, err
	}
	p := &Provider{key: key}
	p.val.Store(client)
	return p, nil
}

func (p *Provider) Get() *mail.Client {
	v := p.val.Load()
	if v == nil {
		return nil
	}
	return v.(*mail.Client)
}

func (p *Provider) Update(cfg Config) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	newDB, err := Init(cfg)
	if err != nil {
		return err
	}

	old := p.Get()
	p.val.Store(newDB)

	if old != nil {
		_ = old.Close()
	}

	return nil
}

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

	return p.Update(c)
}

// todo: 没什么用
func (p *Provider) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	old := p.Get()
	if old == nil {
		return nil
	}
	return old.Close()
}
