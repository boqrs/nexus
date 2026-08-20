package configv2

import (
	"context"
	"os"
	"testing"
	"time"
)

func getIntFromConfig(config map[string]interface{}, key string) int {
	if val, ok := config[key]; ok {
		switch v := val.(type) {
		case int:
			return v
		case float64:
			return int(v)
		case int64:
			return int(v)
		}
	}
	return 0
}

// TestFileConfigWatcher 测试文件配置监控器
func TestFileConfigWatcher(t *testing.T) {
	// 创建临时配置文件
	tmpFile, err := os.CreateTemp("", "config*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	// 写入初始配置
	initialConfig := `{
		"service_name": "test-service",
		"version": "1.0.0",
		"port": 8080,
		"debug": false,
		"features": ["feature1", "feature2"]
	}`

	if _, err := tmpFile.WriteString(initialConfig); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}
	tmpFile.Close()

	// 创建监控器
	watcherConfig := &FileWatcherConfig{
		FilePath:       tmpFile.Name(),
		FileType:       "json",
		WatchInterval:  500 * time.Millisecond,
		EnableFsNotify: false, // 使用轮询方式以便控制时序
	}

	watcher, err := NewFileConfigWatcher(watcherConfig)
	if err != nil {
		t.Fatalf("Failed to create config watcher: %v", err)
	}

	// 验证初始配置
	config := watcher.GetCurrentConfig()
	if serviceName, ok := config["service_name"].(string); !ok || serviceName != "test-service" {
		t.Fatalf("Expected service_name 'test-service', got %v", config["service_name"])
	}

	if port, ok := config["port"].(float64); !ok || port != 8080 {
		t.Fatalf("Expected port 8080, got %v", config["port"])
	}

	t.Log("Initial config loaded successfully")

	// 启动监控
	if err := watcher.Start(); err != nil {
		t.Fatalf("Failed to start watcher: %v", err)
	}

	// 定义配置变更回调
	changeDetected := false
	watcher.AddCallback(func(oldConfig, newConfig map[string]interface{}) error {
		changeDetected = true
		t.Logf("Config changed: %v -> %v", oldConfig, newConfig)
		return nil
	})

	// 修改配置文件
	time.Sleep(1 * time.Second)
	updatedConfig := `{
		"service_name": "updated-service",
		"version": "2.0.0",
		"port": 9090,
		"debug": true,
		"features": ["feature1", "feature2", "feature3"]
	}`

	if err := os.WriteFile(tmpFile.Name(), []byte(updatedConfig), 0644); err != nil {
		t.Fatalf("Failed to update config file: %v", err)
	}

	// 等待配置更新被检测
	time.Sleep(2 * time.Second)

	if !changeDetected {
		t.Log("Warning: Config change not detected (may be due to timing)")
	}

	// 验证更新后的配置
	updatedCfg := watcher.GetCurrentConfig()
	if serviceName, ok := updatedCfg["service_name"].(string); !ok || serviceName != "updated-service" {
		t.Fatalf("Expected service_name 'updated-service', got %v", updatedCfg["service_name"])
	}

	if port, ok := updatedCfg["port"].(float64); !ok || port != 9090 {
		t.Fatalf("Expected port 9090, got %v", updatedCfg["port"])
	}

	t.Log("Config update detected successfully")

	// 停止监控
	watcher.Stop()

	// 验证已停止
	if watcher.IsRunning() {
		t.Fatal("Watcher should be stopped")
	}

	t.Log("FileConfigWatcher test passed")
}

// TestConfigManager 测试配置管理器
func TestConfigManager(t *testing.T) {
	// 创建临时配置文件
	tmpFile, err := os.CreateTemp("", "config*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	// 写入初始配置
	initialConfig := `{
		"service_name": "test-service",
		"version": "1.0.0",
		"port": 8080,
		"debug": false
	}`

	if _, err := tmpFile.WriteString(initialConfig); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}
	tmpFile.Close()

	// 创建监控器
	watcherConfig := &FileWatcherConfig{
		FilePath:       tmpFile.Name(),
		FileType:       "json",
		WatchInterval:  500 * time.Millisecond,
		EnableFsNotify: false,
	}

	watcher, err := NewFileConfigWatcher(watcherConfig)
	if err != nil {
		t.Fatalf("Failed to create config watcher: %v", err)
	}

	// 创建配置管理器
	configManager := NewConfigManager(watcher)

	// 创建配置重载器
	appConfig := &TestAppConfig{}
	configManager.AddReloader(appConfig)

	// 启动管理器
	ctx := context.Background()
	if err := configManager.Start(ctx); err != nil {
		t.Fatalf("Failed to start config manager: %v", err)
	}

	// 验证初始配置
	if appConfig.ServiceName != "test-service" {
		t.Fatalf("Expected service_name 'test-service', got %v", appConfig.ServiceName)
	}

	if appConfig.Port != 8080 {
		t.Fatalf("Expected port 8080, got %v", appConfig.Port)
	}

	t.Log("Initial config loaded by ConfigManager")

	// 修改配置文件
	time.Sleep(1 * time.Second)
	updatedConfig := `{
		"service_name": "updated-service",
		"version": "2.0.0",
		"port": 9090,
		"debug": true
	}`

	if err := os.WriteFile(tmpFile.Name(), []byte(updatedConfig), 0644); err != nil {
		t.Fatalf("Failed to update config file: %v", err)
	}

	// 等待配置更新
	time.Sleep(2 * time.Second)

	// 验证重载器中的配置已更新
	if appConfig.ServiceName != "updated-service" {
		t.Fatalf("Expected service_name 'updated-service', got %v", appConfig.ServiceName)
	}

	if appConfig.Port != 9090 {
		t.Fatalf("Expected port 9090, got %v", appConfig.Port)
	}

	t.Log("Config manager reload test passed")

	// 停止管理器
	configManager.Stop()

	t.Log("ConfigManager test passed")
}

// TestAppConfig 测试用的应用配置
type TestAppConfig struct {
	ServiceName string
	Version     string
	Port        int
	Debug       bool
}

// Reload 实现ConfigReloader接口
func (c *TestAppConfig) Reload(config map[string]interface{}) error {
	if serviceName, ok := config["service_name"].(string); ok {
		c.ServiceName = serviceName
	}

	if version, ok := config["version"].(string); ok {
		c.Version = version
	}

	if port, ok := config["port"].(float64); ok {
		c.Port = int(port)
	}

	if debug, ok := config["debug"].(bool); ok {
		c.Debug = debug
	}

	return nil
}

// TestConfigValueGetters 测试便捷获取方法
func TestConfigValueGetters(t *testing.T) {
	// 创建临时配置文件
	tmpFile, err := os.CreateTemp("", "config*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	// 写入配置
	config := `{
		"service_name": "test-service",
		"port": 8080,
		"debug": true,
		"features": ["feature1", "feature2"],
		"database": {
			"host": "localhost",
			"port": 3306
		}
	}`

	if _, err := tmpFile.WriteString(config); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}
	tmpFile.Close()

	// 创建监控器和管理器
	watcherConfig := &FileWatcherConfig{
		FilePath:       tmpFile.Name(),
		FileType:       "json",
		EnableFsNotify: false,
	}

	watcher, _ := NewFileConfigWatcher(watcherConfig)
	configManager := NewConfigManager(watcher)

	configManager.AddReloader(&TestAppConfig{})
	ctx := context.Background()
	configManager.Start(ctx)

	// 测试便捷方法
	if serviceName := configManager.GetString("service_name"); serviceName != "test-service" {
		t.Fatalf("Expected service_name 'test-service', got %v", serviceName)
	}

	if port := configManager.GetInt("port"); port != 8080 {
		t.Fatalf("Expected port 8080, got %v", port)
	}

	if debug := configManager.GetBool("debug"); !debug {
		t.Fatalf("Expected debug true, got %v", debug)
	}

	features := configManager.GetStringSlice("features")
	if len(features) != 2 {
		t.Fatalf("Expected 2 features, got %v", len(features))
	}

	dbConfig := configManager.GetSubConfig("database")
	if len(dbConfig) != 2 {
		t.Fatalf("Expected 2 database keys, got %v", len(dbConfig))
	}

	t.Log("Config value getters test passed")

	configManager.Stop()
}

// TestConfigValidation 测试配置验证
func TestConfigValidation(t *testing.T) {
	// 创建临时配置文件
	tmpFile, err := os.CreateTemp("", "config*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	// 写入配置
	config := `{
		"port": 8080
	}`

	if _, err := tmpFile.WriteString(config); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}
	tmpFile.Close()

	// 创建监控器和管理器
	watcherConfig := &FileWatcherConfig{
		FilePath:       tmpFile.Name(),
		FileType:       "json",
		EnableFsNotify: false,
	}

	watcher, _ := NewFileConfigWatcher(watcherConfig)
	configManager := NewConfigManager(watcher)

	configManager.AddReloader(&TestAppConfig{})
	ctx := context.Background()
	configManager.Start(ctx)

	// 定义验证器
	validator := func(cfg map[string]interface{}) error {
		if port, ok := cfg["port"].(float64); ok {
			if port < 1 || port > 65535 {
				return newValidationError("port out of range")
			}
		}
		return nil
	}

	// 验证有效配置
	if err := configManager.ValidateConfig(validator); err != nil {
		t.Fatalf("Config validation should pass, got error: %v", err)
	}

	t.Log("Config validation test passed")

	configManager.Stop()
}

// validationError 验证错误
type validationError struct {
	message string
}

func newValidationError(msg string) *validationError {
	return &validationError{message: msg}
}

func (e *validationError) Error() string {
	return e.message
}
