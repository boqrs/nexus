package config

import (
	"testing"
	"time"
)

// TestConfigWatcher 测试ConfigMap监控器
func TestConfigWatcher(t *testing.T) {
	// 注意：这个测试需要在有Kubernetes集群的环境中运行
	// 这里只是一个结构示例

	t.Skip("Skipping test that requires Kubernetes cluster")

	config := &ConfigWatcherConfig{
		Namespace:     "default",
		ConfigMapName: "test-config",
		ConfigKey:     "config.json",
	}

	watcher, err := NewConfigWatcher(config)
	if err != nil {
		t.Fatalf("Failed to create watcher: %v", err)
	}

	// 测试回调
	watcher.AddCallback(func(old, new map[string]interface{}) error {
		return nil
	})

	// 创建配置管理器
	manager := NewConfigManager(watcher)

	// 测试重载器
	testConfig := &ExampleConfig{}
	manager.AddReloader(testConfig)

	// 启动（在真实环境中）
	// ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	// defer cancel()

	// 这里只是测试初始化，不启动实际的监控
	if watcher.namespace != "default" {
		t.Errorf("Expected namespace 'default', got %s", watcher.namespace)
	}

	if watcher.configMapName != "test-config" {
		t.Errorf("Expected configmap name 'test-config', got %s", watcher.configMapName)
	}

	t.Log("ConfigWatcher basic initialization test passed")
}

// TestConfigReloader 测试配置重载器
func TestConfigReloader(t *testing.T) {
	config := &ExampleConfig{
		ServiceName: "test-service",
		Version:     "1.0.0",
		Timeout:     30 * time.Second,
		Features:    []string{"test"},
	}

	newConfig := map[string]interface{}{
		"service_name": "updated-service",
		"version":      "2.0.0",
		"timeout":      "60s",
		"features":     []interface{}{"tracing", "metrics"},
	}

	err := config.Reload(newConfig)
	if err != nil {
		t.Fatalf("Failed to reload config: %v", err)
	}

	if config.ServiceName != "updated-service" {
		t.Errorf("Expected service name 'updated-service', got %s", config.ServiceName)
	}

	if config.Version != "2.0.0" {
		t.Errorf("Expected version '2.0.0', got %s", config.Version)
	}

	if config.Timeout != 60*time.Second {
		t.Errorf("Expected timeout 60s, got %v", config.Timeout)
	}

	if len(config.Features) != 2 || config.Features[0] != "tracing" {
		t.Errorf("Expected features ['tracing', 'metrics'], got %v", config.Features)
	}

	t.Log("ConfigReloader test passed")
}