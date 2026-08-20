package configv2

import (
	"context"
	"fmt"
	"log"
	"sync"
)

// ConfigReloader 接口：任何希望自动接收配置更新的结构体都应实现此接口
type ConfigReloader interface {
	Reload(config map[string]interface{}) error
}

// ConfigManager 配置管理器
type ConfigManager struct {
	watcher   *FileConfigWatcher
	reloaders []ConfigReloader
	mu        sync.RWMutex
}

// NewConfigManager 创建配置管理器
func NewConfigManager(watcher *FileConfigWatcher) *ConfigManager {
	return &ConfigManager{
		watcher:   watcher,
		reloaders: make([]ConfigReloader, 0),
	}
}

// AddReloader 添加配置重载器
func (cm *ConfigManager) AddReloader(reloader ConfigReloader) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.reloaders = append(cm.reloaders, reloader)
}

// Start 启动配置管理器
func (cm *ConfigManager) Start(ctx context.Context) error {
	// 1. 【关键】在启动监控前，先获取当前配置并初始化所有 Reloader
	currentConfig := cm.watcher.GetCurrentConfig()

	cm.mu.RLock()
	for _, reloader := range cm.reloaders {
		if err := reloader.Reload(currentConfig); err != nil {
			cm.mu.RUnlock()
			return fmt.Errorf("failed to initialize config for reloader: %w", err)
		}
	}
	cm.mu.RUnlock()

	log.Println("Config managers initialized with current file content")

	// 2. 注册一个内部回调，用于在文件变更时通知所有 Reloader
	cm.watcher.AddCallback(func(oldConfig, newConfig map[string]interface{}) error {
		return cm.notifyReloaders(newConfig)
	})

	// 3. 启动底层文件监控
	return cm.watcher.Start()
}

// notifyReloaders 通知所有重载器更新配置
func (cm *ConfigManager) notifyReloaders(newConfig map[string]interface{}) error {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	var lastErr error
	for _, reloader := range cm.reloaders {
		if err := reloader.Reload(newConfig); err != nil {
			log.Printf("Error reloading config for specific module: %v", err)
			lastErr = err
		}
	}
	return lastErr
}

// Stop 停止配置管理器
func (cm *ConfigManager) Stop() {
	cm.watcher.Stop()
}

// GetCurrentConfig 获取当前原始配置 Map
func (cm *ConfigManager) GetCurrentConfig() map[string]interface{} {
	return cm.watcher.GetCurrentConfig()
}

// --- 以下是测试需要的便捷方法 ---

// ValidateConfig 验证当前配置
func (cm *ConfigManager) ValidateConfig(validator func(map[string]interface{}) error) error {
	config := cm.watcher.GetCurrentConfig()
	return validator(config)
}

// GetString 获取字符串配置
func (cm *ConfigManager) GetString(key string) string {
	config := cm.watcher.GetCurrentConfig()
	if val, ok := config[key].(string); ok {
		return val
	}
	return ""
}

// GetInt 获取整数配置
func (cm *ConfigManager) GetInt(key string) int {
	config := cm.watcher.GetCurrentConfig()
	if val, ok := config[key].(float64); ok {
		return int(val)
	}
	if val, ok := config[key].(int); ok {
		return val
	}
	return 0
}

// GetBool 获取布尔配置
func (cm *ConfigManager) GetBool(key string) bool {
	config := cm.watcher.GetCurrentConfig()
	if val, ok := config[key].(bool); ok {
		return val
	}
	return false
}

// GetStringSlice 获取字符串切片配置
func (cm *ConfigManager) GetStringSlice(key string) []string {
	config := cm.watcher.GetCurrentConfig()
	if val, ok := config[key].([]interface{}); ok {
		result := make([]string, len(val))
		for i, v := range val {
			if s, ok := v.(string); ok {
				result[i] = s
			}
		}
		return result
	}
	if val, ok := config[key].([]string); ok {
		return val
	}
	return []string{}
}

// GetSubConfig 获取子配置 Map
func (cm *ConfigManager) GetSubConfig(key string) map[string]interface{} {
	config := cm.watcher.GetCurrentConfig()
	if val, ok := config[key].(map[string]interface{}); ok {
		return val
	}
	return map[string]interface{}{}
}
