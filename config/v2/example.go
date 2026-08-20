package configv2

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/mitchellh/mapstructure"
)

// ExampleAppConfig 应用配置示例
type ExampleAppConfig struct {
	ServiceName string        `mapstructure:"service_name"`
	Version     string        `mapstructure:"version"`
	Port        int           `mapstructure:"port"`
	Debug       bool          `mapstructure:"debug"`
	Timeout     time.Duration `mapstructure:"timeout"`
	Features    []string      `mapstructure:"features"`
}

// Reload 实现 ConfigReloader 接口
// 【优雅点】：使用 mapstructure 自动解码，无需手动类型断言
func (c *ExampleAppConfig) Reload(config map[string]interface{}) error {
	// DecodeHook 可以处理一些特殊的类型转换，比如 string -> time.Duration
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Metadata:         nil,
		Result:           c,
		WeaklyTypedInput: true, // 允许弱类型转换，例如 float64 转 int
		DecodeHook: mapstructure.ComposeDecodeHookFunc(
			mapstructure.StringToTimeDurationHookFunc(),
		),
	})
	if err != nil {
		return fmt.Errorf("failed to create decoder: %w", err)
	}

	if err := decoder.Decode(config); err != nil {
		return fmt.Errorf("failed to decode config: %w", err)
	}

	log.Printf("Config reloaded successfully: Service=%s, Port=%d, Debug=%v",
		c.ServiceName, c.Port, c.Debug)
	return nil
}

// ExampleUsage 优雅的使用示例
func ExampleUsage() {
	log.Println("Starting elegant config watcher example...")

	// 1. 创建监控器
	watcherConfig := &FileWatcherConfig{
		FilePath:       "config.yaml", // 推荐使用 YAML，可读性更好
		FileType:       "yaml",
		WatchInterval:  1 * time.Second,
		EnableFsNotify: true,
	}

	watcher, err := NewFileConfigWatcher(watcherConfig)
	if err != nil {
		log.Fatalf("Failed to create config watcher: %v", err)
	}

	// 2. 创建配置管理器
	configManager := NewConfigManager(watcher)

	// 3. 创建【空】的业务配置结构体
	// 注意：这里不需要填任何默认值！
	appConfig := &ExampleAppConfig{}

	// 4. 注册重载器
	configManager.AddReloader(appConfig)

	// 5. 启动管理器
	// 【关键点】：Start() 内部会自动读取文件内容并调用 appConfig.Reload()
	// 此时 appConfig 已经被填充为文件中的真实数据
	ctx := context.Background()
	if err := configManager.Start(ctx); err != nil {
		log.Fatalf("Failed to start config manager: %v", err)
	}

	// 6. 直接使用配置
	log.Printf("Application started with config: %+v", appConfig)
	log.Printf("Connecting to database at port %d...", appConfig.Port)

	// 模拟运行
	select {
	case <-time.After(5 * time.Minute):
	}

	configManager.Stop()
	log.Println("Config watcher stopped")
}

// AdvancedExample 高级示例
func AdvancedExample() {
	watcherConfig := &FileWatcherConfig{
		FilePath:       "config.yaml",
		FileType:       "yaml",
		EnableFsNotify: true,
	}

	watcher, err := NewFileConfigWatcher(watcherConfig)
	if err != nil {
		log.Fatalf("Failed to create watcher: %v", err)
	}

	configManager := NewConfigManager(watcher)

	// 同样只需要空结构体
	appConfig := &ExampleAppConfig{}
	configManager.AddReloader(appConfig)

	ctx := context.Background()
	if err := configManager.Start(ctx); err != nil {
		log.Fatalf("Failed to start: %v", err)
	}
	defer configManager.Stop()

	// 业务逻辑中直接使用 appConfig 指针即可，它会随文件变化自动更新
	fmt.Printf("Current Service: %s\n", appConfig.ServiceName)
}
