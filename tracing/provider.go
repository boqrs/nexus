package tracing

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/mitchellh/mapstructure"
)

// Provider 是 tracing 模块对外的标准入口。
// 它实现了 comm/config.ConfigReloader 接口，支持配置热更新。
type Provider struct {
	manager *ProviderManager
	cfg     *Config
	mu      sync.Mutex
}

// NewProvider 创建 Tracing Provider。
// 注意：由于 OTel 全局单例的限制，多次调用 NewProvider 只会初始化一次全局 Provider，
// 但会返回新的 Provider 实例用于管理配置引用和热更新。
func NewProvider(cfg *Config) (*Provider, error) {
	if cfg == nil {
		return nil, fmt.Errorf("tracing config is nil")
	}

	// 初始化全局 Manager (内部有 sync.Once 保护)
	manager, err := InitGlobalTracer(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to init global tracer: %w", err)
	}

	// 如果之前已初始化，InitGlobalTracer 可能返回 nil manager 但无 error，
	// 此时需获取现有的全局 Manager
	if manager == nil {
		manager = GetManager()
		if manager == nil {
			return nil, fmt.Errorf("tracing manager is nil after init")
		}
	}

	return &Provider{
		manager: manager,
		cfg:     cfg,
	}, nil
}

// Reload 实现 comm/config.ConfigReloader 接口。
// 当配置文件变更时，ConfigManager 会调用此方法。
func (p *Provider) Reload(config map[string]interface{}) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	log.Println("Tracing Provider: Reloading config...")

	// 1. 提取 tracing 子配置
	targetConfig := config
	if sub, ok := config["tracing"].(map[string]interface{}); ok {
		targetConfig = sub
	} else if sub, ok := config["tracing"].(map[interface{}]interface{}); ok {
		// 兼容 YAML 解析出的 map[interface{}]interface{}
		flatSub := make(map[string]interface{})
		for k, v := range sub {
			if ks, ok := k.(string); ok {
				flatSub[ks] = v
			}
		}
		targetConfig = flatSub
	}

	// 2. 使用 mapstructure 解码配置 (支持 time.Duration 等复杂类型)
	var newCfg Config
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Result:           &newCfg,
		WeaklyTypedInput: true,
		DecodeHook: mapstructure.ComposeDecodeHookFunc(
			mapstructure.StringToTimeDurationHookFunc(),
		),
	})
	if err != nil {
		return fmt.Errorf("failed to create decoder: %w", err)
	}

	if err := decoder.Decode(targetConfig); err != nil {
		return fmt.Errorf("failed to decode tracing config: %w", err)
	}

	// 3. 应用配置热更新
	if err := p.manager.UpdateConfig(&newCfg); err != nil {
		return fmt.Errorf("failed to update tracing config: %w", err)
	}

	// 4. 更新本地持有的配置引用
	p.cfg = &newCfg
	return nil
}

// Close 优雅关闭 Tracing Provider，确保未发送的 Span 被导出。
func (p *Provider) Close() error {
	if p.manager != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return p.manager.Shutdown(ctx)
	}
	return nil
}
