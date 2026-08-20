package config

import (
	"context"
	"fmt"
	"time"

	stdlog "log"

	"github.com/boqrs/nexus/log"
)

// ExampleConfig 示例配置结构
type ExampleConfig struct {
	ServiceName string        `json:"service_name"`
	Version     string        `json:"version"`
	Database    DBConfig      `json:"database"`
	Log         log.LogConfig `json:"log"`
	Timeout     time.Duration `json:"timeout"`
	Features    []string      `json:"features"`
}

// Reload 实现ConfigReloader接口
func (c *ExampleConfig) Reload(config map[string]interface{}) error {
	stdlog.Printf("Reloading example config: %v", config)

	// 这里可以实现配置的热更新逻辑
	// 例如：更新数据库连接、日志级别等

	if serviceName, ok := config["service_name"].(string); ok {
		c.ServiceName = serviceName
		stdlog.Printf("Updated service name: %s", serviceName)
	}

	if version, ok := config["version"].(string); ok {
		c.Version = version
		stdlog.Printf("Updated version: %s", version)
	}

	if timeoutStr, ok := config["timeout"].(string); ok {
		if timeout, err := time.ParseDuration(timeoutStr); err == nil {
			c.Timeout = timeout
			stdlog.Printf("Updated timeout: %v", timeout)
		}
	}

	if features, ok := config["features"].([]interface{}); ok {
		c.Features = make([]string, len(features))
		for i, f := range features {
			if feature, ok := f.(string); ok {
				c.Features[i] = feature
			}
		}
		stdlog.Printf("Updated features: %v", c.Features)
	}

	return nil
}

// ExampleUsage ConfigMap监控器使用示例
func ExampleUsage() {
	// 注意：这个示例需要在Kubernetes集群中运行
	// 这里只是展示API使用方法

	stdlog.Println("Starting ConfigMap watcher example...")

	// 创建ConfigMap监控器配置
	watcherConfig := &ConfigWatcherConfig{
		Namespace:     "default",     // ConfigMap所在命名空间
		ConfigMapName: "app-config",  // ConfigMap名称
		ConfigKey:     "config.json", // 配置在ConfigMap中的key
		// KubeConfig: "/path/to/kubeconfig", // 可选：外部kubeconfig路径
	}

	// 创建监控器
	watcher, err := NewConfigWatcher(watcherConfig)
	if err != nil {
		stdlog.Fatalf("Failed to create config watcher: %v", err)
	}

	// 创建配置管理器
	configManager := NewConfigManager(watcher)

	// 创建示例配置实例
	exampleConfig := &ExampleConfig{
		ServiceName: "example-service",
		Version:     "1.0.0",
		Timeout:     30 * time.Second,
		Features:    []string{"feature1", "feature2"},
	}

	// 添加配置重载器
	configManager.AddReloader(exampleConfig)

	// 启动配置管理器
	ctx := context.Background()
	if err := configManager.Start(ctx); err != nil {
		stdlog.Fatalf("Failed to start config manager: %v", err)
	}

	stdlog.Println("ConfigMap watcher started successfully")
	stdlog.Printf("Current config: %v", configManager.GetCurrentConfig())

	// 模拟运行
	fmt.Println("ConfigMap watcher is running...")
	fmt.Println("You can update the ConfigMap in Kubernetes to see config changes")
	fmt.Println("Press Ctrl+C to stop")

	// 等待中断信号
	<-ctx.Done()
	configManager.Stop()
}

// ExampleConfigMapYAML ConfigMap的YAML示例
const ExampleConfigMapYAML = `
apiVersion: v1
kind: ConfigMap
metadata:
  name: app-config
  namespace: default
data:
  config.json: |
    {
      "service_name": "my-microservice",
      "version": "2.1.0",
      "database": {
        "driver": "mysql",
        "host": "mysql-service",
        "port": 3306,
        "username": "app",
        "password": "secret",
        "database": "app_db",
        "max_open": 20,
        "max_idle": 10
      },
      "log": {
        "level": "info",
        "dir": "/var/log/app"
      },
      "timeout": "60s",
      "features": ["tracing", "metrics", "healthcheck"]
    }`

// PrintExampleConfigMap 打印ConfigMap示例
func PrintExampleConfigMap() {
	fmt.Println("Example ConfigMap YAML:")
	fmt.Println(ExampleConfigMapYAML)
}