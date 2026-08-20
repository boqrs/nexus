package configv2

import (
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

// FileConfigWatcher 文件配置监控器
type FileConfigWatcher struct {
	viper          *viper.Viper
	filePath       string
	fileType       string
	currentConfig  map[string]interface{}
	callbacks      []ConfigChangeCallback
	stopCh         chan struct{}
	mu             sync.RWMutex
	running        bool
	watchInterval  time.Duration
	lastModTime    time.Time
	enableFsNotify bool
}

// ConfigChangeCallback 配置变更回调
type ConfigChangeCallback func(oldConfig, newConfig map[string]interface{}) error

// FileWatcherConfig 监控器配置
type FileWatcherConfig struct {
	FilePath       string
	FileType       string
	WatchInterval  time.Duration
	EnableFsNotify bool
	OnConfigChange ConfigChangeCallback
}

// NewFileConfigWatcher 创建监控器
func NewFileConfigWatcher(config *FileWatcherConfig) (*FileConfigWatcher, error) {
	if config.FilePath == "" {
		return nil, fmt.Errorf("config file path cannot be empty")
	}
	if config.FileType == "" {
		config.FileType = "json"
	}
	if config.WatchInterval == 0 {
		config.WatchInterval = 1 * time.Second
	}

	v := viper.New()
	v.SetConfigFile(config.FilePath)
	v.SetConfigType(config.FileType)

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	watcher := &FileConfigWatcher{
		viper:          v,
		filePath:       config.FilePath,
		fileType:       config.FileType,
		currentConfig:  v.AllSettings(),
		callbacks:      make([]ConfigChangeCallback, 0),
		stopCh:         make(chan struct{}),
		watchInterval:  config.WatchInterval,
		enableFsNotify: config.EnableFsNotify,
	}

	if config.OnConfigChange != nil {
		watcher.callbacks = append(watcher.callbacks, config.OnConfigChange)
	}

	if fileInfo, err := os.Stat(config.FilePath); err == nil {
		watcher.lastModTime = fileInfo.ModTime()
	}

	return watcher, nil
}

// AddCallback 添加回调
func (w *FileConfigWatcher) AddCallback(callback ConfigChangeCallback) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.callbacks = append(w.callbacks, callback)
}

// Start 启动监控
func (w *FileConfigWatcher) Start() error {
	w.mu.Lock()
	if w.running {
		w.mu.Unlock()
		return fmt.Errorf("watcher is already running")
	}
	w.running = true
	w.mu.Unlock()

	if w.enableFsNotify {
		w.viper.OnConfigChange(func(e fsnotify.Event) {
			log.Printf("Config file changed: %s", e.Name)
			w.reloadConfig()
		})
		w.viper.WatchConfig()
		log.Printf("File config watcher started (fsnotify) - file=%s", w.filePath)
	} else {
		go w.pollWatchLoop()
		log.Printf("File config watcher started (polling) - file=%s interval=%v", w.filePath, w.watchInterval)
	}

	return nil
}

// Stop 停止监控
func (w *FileConfigWatcher) Stop() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if !w.running {
		return
	}
	w.running = false
	close(w.stopCh)
	log.Println("File config watcher stopped")
}

func (w *FileConfigWatcher) IsRunning() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.running
}

// GetCurrentConfig 获取当前配置副本
func (w *FileConfigWatcher) GetCurrentConfig() map[string]interface{} {
	w.mu.RLock()
	defer w.mu.RUnlock()
	config := make(map[string]interface{})
	for k, v := range w.currentConfig {
		config[k] = v
	}
	return config
}

// pollWatchLoop 轮询循环
func (w *FileConfigWatcher) pollWatchLoop() {
	ticker := time.NewTicker(w.watchInterval)
	defer ticker.Stop()
	for {
		select {
		case <-w.stopCh:
			return
		case <-ticker.C:
			w.reloadConfig()
		}
	}
}

// reloadConfig 重新加载配置（含重试机制）
func (w *FileConfigWatcher) reloadConfig() {
	var err error
	// 重试两次以应对文件写入竞态
	for i := 0; i < 2; i++ {
		err = w.viper.ReadInConfig()
		if err == nil {
			break
		}
		if i < 1 {
			time.Sleep(50 * time.Millisecond)
		}
	}

	if err != nil {
		log.Printf("Failed to reload config after retry: %v", err)
		return
	}

	if fileInfo, err := os.Stat(w.filePath); err == nil {
		w.mu.Lock()
		w.lastModTime = fileInfo.ModTime()
		w.mu.Unlock()
	}

	w.mu.Lock()
	newConfig := w.viper.AllSettings()
	oldConfig := w.currentConfig

	if configEqual(oldConfig, newConfig) {
		w.mu.Unlock()
		return
	}

	w.currentConfig = newConfig
	w.mu.Unlock()

	log.Printf("Config file updated - file=%s", w.filePath)
	w.triggerCallbacks(oldConfig, newConfig)
}

// triggerCallbacks 触发回调
func (w *FileConfigWatcher) triggerCallbacks(oldConfig, newConfig map[string]interface{}) {
	w.mu.RLock()
	callbacks := make([]ConfigChangeCallback, len(w.callbacks))
	copy(callbacks, w.callbacks)
	w.mu.RUnlock()

	for _, callback := range callbacks {
		if err := callback(oldConfig, newConfig); err != nil {
			log.Printf("Config change callback failed: %v", err)
		}
	}
}

// 辅助函数：比较配置是否变化
func configEqual(oldConfig, newConfig map[string]interface{}) bool {
	if len(oldConfig) != len(newConfig) {
		return false
	}
	for k, v := range oldConfig {
		newV, exists := newConfig[k]
		if !exists || !deepEqual(v, newV) {
			return false
		}
	}
	return true
}

func deepEqual(a, b interface{}) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	switch aVal := a.(type) {
	case map[string]interface{}:
		bVal, ok := b.(map[string]interface{})
		if !ok || len(aVal) != len(bVal) {
			return false
		}
		for k, av := range aVal {
			bv, exists := bVal[k]
			if !exists || !deepEqual(av, bv) {
				return false
			}
		}
		return true
	case []interface{}:
		bVal, ok := b.([]interface{})
		if !ok || len(aVal) != len(bVal) {
			return false
		}
		for i, av := range aVal {
			if !deepEqual(av, bVal[i]) {
				return false
			}
		}
		return true
	default:
		return a == b
	}
}
