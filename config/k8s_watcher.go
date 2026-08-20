package config

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"log"
)

// ConfigWatcher ConfigMap配置监控器
type ConfigWatcher struct {
	client        kubernetes.Interface
	namespace     string
	configMapName string
	configKey     string
	currentConfig map[string]interface{}
	callbacks     []ConfigChangeCallback
	stopCh        chan struct{}
	mu            sync.RWMutex
	running       bool
}

// ConfigChangeCallback 配置变更回调函数类型
type ConfigChangeCallback func(oldConfig, newConfig map[string]interface{}) error

// ConfigWatcherConfig 监控器配置
type ConfigWatcherConfig struct {
	// Kubernetes配置
	KubeConfig    string // kubeconfig文件路径，留空则使用集群内配置
	Namespace     string // ConfigMap所在命名空间
	ConfigMapName string // ConfigMap名称
	ConfigKey     string // 配置在ConfigMap中的key

	// 监控配置
	ResyncPeriod time.Duration // 重新同步周期
	RetryDelay   time.Duration // 重试延迟

	// 回调配置
	OnConfigChange ConfigChangeCallback // 配置变更回调
}

// NewConfigWatcher 创建ConfigMap监控器
func NewConfigWatcher(config *ConfigWatcherConfig) (*ConfigWatcher, error) {
	if config.Namespace == "" {
		config.Namespace = "default"
	}
	if config.ResyncPeriod == 0 {
		config.ResyncPeriod = 5 * time.Minute
	}
	if config.RetryDelay == 0 {
		config.RetryDelay = 5 * time.Second
	}

	// 创建Kubernetes客户端
	var k8sConfig *rest.Config
	var err error

	if config.KubeConfig != "" {
		// 使用外部kubeconfig
		k8sConfig, err = clientcmd.BuildConfigFromFlags("", config.KubeConfig)
	} else {
		// 使用集群内配置
		k8sConfig, err = rest.InClusterConfig()
	}
	if err != nil {
		return nil, fmt.Errorf("failed to create k8s config: %w", err)
	}

	client, err := kubernetes.NewForConfig(k8sConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create k8s client: %w", err)
	}

	watcher := &ConfigWatcher{
		client:        client,
		namespace:     config.Namespace,
		configMapName: config.ConfigMapName,
		configKey:     config.ConfigKey,
		currentConfig: make(map[string]interface{}),
		callbacks:     make([]ConfigChangeCallback, 0),
		stopCh:        make(chan struct{}),
	}

	// 添加默认回调
	if config.OnConfigChange != nil {
		watcher.callbacks = append(watcher.callbacks, config.OnConfigChange)
	}

	return watcher, nil
}

// AddCallback 添加配置变更回调
func (w *ConfigWatcher) AddCallback(callback ConfigChangeCallback) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.callbacks = append(w.callbacks, callback)
}

// Start 启动监控
func (w *ConfigWatcher) Start(ctx context.Context) error {
	w.mu.Lock()
	if w.running {
		w.mu.Unlock()
		return fmt.Errorf("watcher is already running")
	}
	w.running = true
	w.mu.Unlock()

	// 首次加载配置
	if err := w.loadInitialConfig(ctx); err != nil {
		log.Printf("Failed to load initial config: %v", err)
		w.mu.Lock()
		w.running = false
		w.mu.Unlock()
		return err
	}

	// 启动监控协程
	go w.watchLoop(ctx)

	log.Printf("ConfigMap watcher started namespace=%s configmap=%s key=%s",
		w.namespace, w.configMapName, w.configKey)

	return nil
}

// Stop 停止监控
func (w *ConfigWatcher) Stop() {
	w.mu.Lock()
	defer w.mu.Unlock()

	if !w.running {
		return
	}

	w.running = false
	close(w.stopCh)

	log.Println("ConfigMap watcher stopped")
}

// IsRunning 检查是否正在运行
func (w *ConfigWatcher) IsRunning() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.running
}

// GetCurrentConfig 获取当前配置
func (w *ConfigWatcher) GetCurrentConfig() map[string]interface{} {
	w.mu.RLock()
	defer w.mu.RUnlock()

	// 返回配置的副本
	config := make(map[string]interface{})
	for k, v := range w.currentConfig {
		config[k] = v
	}
	return config
}

// loadInitialConfig 加载初始配置
func (w *ConfigWatcher) loadInitialConfig(ctx context.Context) error {
	cm, err := w.client.CoreV1().ConfigMaps(w.namespace).Get(ctx, w.configMapName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get ConfigMap: %w", err)
	}

	return w.processConfigMap(cm)
}

// watchLoop 监控循环
func (w *ConfigWatcher) watchLoop(ctx context.Context) {
	for {
		select {
		case <-w.stopCh:
			return
		case <-ctx.Done():
			return
		default:
			if err := w.watchConfigMap(ctx); err != nil {
				log.Printf("ConfigMap watch error: %v", err)
				// 等待重试
				select {
				case <-w.stopCh:
					return
				case <-time.After(5 * time.Second):
					continue
				}
			}
		}
	}
}

// watchConfigMap 监控ConfigMap变化
func (w *ConfigWatcher) watchConfigMap(ctx context.Context) error {
	watcher, err := w.client.CoreV1().ConfigMaps(w.namespace).Watch(ctx, metav1.ListOptions{
		FieldSelector: fmt.Sprintf("metadata.name=%s", w.configMapName),
	})
	if err != nil {
		return fmt.Errorf("failed to watch ConfigMap: %w", err)
	}
	defer watcher.Stop()

	log.Printf("Started watching ConfigMap namespace=%s name=%s", w.namespace, w.configMapName)

	for {
		select {
		case <-w.stopCh:
			return nil
		case <-ctx.Done():
			return nil
		case event, ok := <-watcher.ResultChan():
			if !ok {
				log.Println("ConfigMap watch channel closed")
				return fmt.Errorf("watch channel closed")
			}

			switch event.Type {
			case watch.Added, watch.Modified:
				cm, ok := event.Object.(*corev1.ConfigMap)
				if !ok {
					log.Println("Received non-ConfigMap object")
					continue
				}

				if err := w.processConfigMap(cm); err != nil {
					log.Printf("Failed to process ConfigMap: %v", err)
				}
			case watch.Deleted:
				log.Printf("ConfigMap deleted namespace=%s name=%s", w.namespace, w.configMapName)
			case watch.Error:
				log.Printf("ConfigMap watch error event: %v", event.Object)
			}
		}
	}
}

// processConfigMap 处理ConfigMap
func (w *ConfigWatcher) processConfigMap(cm *corev1.ConfigMap) error {
	data, exists := cm.Data[w.configKey]
	if !exists {
		return fmt.Errorf("config key '%s' not found in ConfigMap", w.configKey)
	}

	var newConfig map[string]interface{}
	if err := json.Unmarshal([]byte(data), &newConfig); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}

	w.mu.Lock()
	oldConfig := w.currentConfig
	w.currentConfig = newConfig
	w.mu.Unlock()

	// 触发回调
	for _, callback := range w.callbacks {
		if err := callback(oldConfig, newConfig); err != nil {
			log.Printf("Config change callback failed: %v", err)
		}
	}

	log.Printf("ConfigMap updated namespace=%s name=%s key=%s",
		w.namespace, w.configMapName, w.configKey)

	return nil
}

// ConfigReloader 配置重载器接口
type ConfigReloader interface {
	Reload(config map[string]interface{}) error
}

// ConfigManager 配置管理器
type ConfigManager struct {
	watcher  *ConfigWatcher
	reloaders []ConfigReloader
	mu       sync.RWMutex
}

// NewConfigManager 创建配置管理器
func NewConfigManager(watcher *ConfigWatcher) *ConfigManager {
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
	// 添加配置变更回调
	cm.watcher.AddCallback(func(oldConfig, newConfig map[string]interface{}) error {
		cm.mu.RLock()
		reloaders := make([]ConfigReloader, len(cm.reloaders))
		copy(reloaders, cm.reloaders)
		cm.mu.RUnlock()

		for _, reloader := range reloaders {
			if err := reloader.Reload(newConfig); err != nil {
				log.Printf("Config reloader failed: %v", err)
				// 继续处理其他重载器
			}
		}
		return nil
	})

	return cm.watcher.Start(ctx)
}

// Stop 停止配置管理器
func (cm *ConfigManager) Stop() {
	cm.watcher.Stop()
}

// GetCurrentConfig 获取当前配置
func (cm *ConfigManager) GetCurrentConfig() map[string]interface{} {
	return cm.watcher.GetCurrentConfig()
}