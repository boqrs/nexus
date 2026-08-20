package tracing

import (
	"fmt"
	"log"
	"time"

	"github.com/mitchellh/mapstructure"
)

// Config 链路追踪配置
// Config 链路追踪配置
type Config struct {
	Enabled            bool              `yaml:"enabled" mapstructure:"enabled"`
	ServiceName        string            `yaml:"service_name" mapstructure:"service_name"`
	ServiceVersion     string            `yaml:"service_version" mapstructure:"service_version"`
	SampleRate         float64           `yaml:"sample_rate" mapstructure:"sample_rate"`
	ResourceAttributes map[string]string `yaml:"resource_attributes" mapstructure:"resource_attributes"`
	Exporter           ExporterConfig    `yaml:"exporter" mapstructure:"exporter"`
}

// ExporterConfig 导出器配置
type ExporterConfig struct {
	Type          string        `yaml:"type" mapstructure:"type"` // "otlp_grpc"
	Endpoint      string        `yaml:"endpoint" mapstructure:"endpoint"`
	Authorization string        `yaml:"authorization" mapstructure:"authorization"` // 可选，用于鉴权，例如 ARMS 的 License Key
	Insecure      bool          `yaml:"insecure" mapstructure:"insecure"`
	Timeout       time.Duration `yaml:"timeout" mapstructure:"timeout"`
	// Headers 用于传递鉴权信息，例如阿里云 ARMS 的 License Key
	// 格式: map[string]string{"x-arms-license-key": "your_key"}
}

// Reload 实现 config/v2 的 ConfigReloader 接口
func (c *Config) Reload(config map[string]interface{}) error {
	log.Println("Tracing: Detecting config change...")

	// 1. 解码配置
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Result:           c,
		WeaklyTypedInput: true,
		DecodeHook: mapstructure.ComposeDecodeHookFunc(
			mapstructure.StringToTimeDurationHookFunc(),
		),
	})
	if err != nil {
		return fmt.Errorf("failed to create decoder: %w", err)
	}

	// 处理嵌套配置 (假设配置在 "tracing" key 下，如果在根节点则直接传 config)
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

	if err := decoder.Decode(targetConfig); err != nil {
		return fmt.Errorf("failed to decode tracing config: %w", err)
	}

	// 2. 应用配置到全局 Manager
	manager := GetManager()
	if manager == nil {
		// 首次启动：初始化全局 Tracer
		log.Println("Tracing: Initializing global provider for the first time...")
		_, err := InitGlobalTracer(c)
		return err
	}

	// 热更新：动态调整采样率或重建 Provider
	log.Printf("Tracing: Updating config (SampleRate: %.2f, Endpoint: %s)", c.SampleRate, c.Exporter.Endpoint)
	return manager.UpdateConfig(c)
}
