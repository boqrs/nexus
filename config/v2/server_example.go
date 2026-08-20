package configv2

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"
)

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Driver   string `json:"driver"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	Database string `json:"database"`
}

// LogConfig 日志配置
type LogConfig struct {
	Level  string `json:"level"`
	Format string `json:"format"`
	Output string `json:"output"`
}

// CacheConfig 缓存配置
type CacheConfig struct {
	Enabled bool   `json:"enabled"`
	TTL     string `json:"ttl"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	ServiceName string
	Version     string
	Port        int
	Debug       bool
	Timeout     time.Duration
	Features    []string
	Database    DatabaseConfig
	Log         LogConfig
	Cache       CacheConfig
}

// Reload 实现ConfigReloader接口，用于热更新
func (c *ServerConfig) Reload(config map[string]interface{}) error {
	log.Println("=== Starting ServerConfig Reload ===")

	// 基本配置
	if serviceName, ok := config["service_name"].(string); ok {
		if serviceName != c.ServiceName {
			log.Printf("ServiceName updated: %s -> %s", c.ServiceName, serviceName)
			c.ServiceName = serviceName
		}
	}

	if version, ok := config["version"].(string); ok {
		if version != c.Version {
			log.Printf("Version updated: %s -> %s", c.Version, version)
			c.Version = version
		}
	}

	if port, ok := config["port"].(float64); ok {
		newPort := int(port)
		if newPort != c.Port {
			log.Printf("Port updated: %d -> %d", c.Port, newPort)
			c.Port = newPort
		}
	}

	if debug, ok := config["debug"].(bool); ok {
		if debug != c.Debug {
			log.Printf("Debug updated: %v -> %v", c.Debug, debug)
			c.Debug = debug
		}
	}

	if timeoutStr, ok := config["timeout"].(string); ok {
		if timeout, err := time.ParseDuration(timeoutStr); err == nil {
			if timeout != c.Timeout {
				log.Printf("Timeout updated: %v -> %v", c.Timeout, timeout)
				c.Timeout = timeout
			}
		}
	}

	// Features列表
	if features, ok := config["features"].([]interface{}); ok {
		newFeatures := make([]string, len(features))
		for i, f := range features {
			if feature, ok := f.(string); ok {
				newFeatures[i] = feature
			}
		}
		if fmt.Sprintf("%v", newFeatures) != fmt.Sprintf("%v", c.Features) {
			log.Printf("Features updated: %v -> %v", c.Features, newFeatures)
			c.Features = newFeatures
		}
	}

	// 数据库配置
	if dbCfg, ok := config["database"].(map[string]interface{}); ok {
		if driver, ok := dbCfg["driver"].(string); ok && driver != c.Database.Driver {
			log.Printf("Database driver updated: %s -> %s", c.Database.Driver, driver)
			c.Database.Driver = driver
		}
		if host, ok := dbCfg["host"].(string); ok && host != c.Database.Host {
			log.Printf("Database host updated: %s -> %s", c.Database.Host, host)
			c.Database.Host = host
		}
		if port, ok := dbCfg["port"].(float64); ok {
			newPort := int(port)
			if newPort != c.Database.Port {
				log.Printf("Database port updated: %d -> %d", c.Database.Port, newPort)
				c.Database.Port = newPort
			}
		}
		if username, ok := dbCfg["username"].(string); ok && username != c.Database.Username {
			log.Printf("Database username updated: %s -> %s", c.Database.Username, username)
			c.Database.Username = username
		}
		if password, ok := dbCfg["password"].(string); ok && password != c.Database.Password {
			log.Printf("Database password updated")
			c.Database.Password = password
		}
		if database, ok := dbCfg["database"].(string); ok && database != c.Database.Database {
			log.Printf("Database name updated: %s -> %s", c.Database.Database, database)
			c.Database.Database = database
		}
	}

	// 日志配置
	if logCfg, ok := config["log"].(map[string]interface{}); ok {
		if level, ok := logCfg["level"].(string); ok && level != c.Log.Level {
			log.Printf("Log level updated: %s -> %s", c.Log.Level, level)
			c.Log.Level = level
		}
		if format, ok := logCfg["format"].(string); ok && format != c.Log.Format {
			log.Printf("Log format updated: %s -> %s", c.Log.Format, format)
			c.Log.Format = format
		}
		if output, ok := logCfg["output"].(string); ok && output != c.Log.Output {
			log.Printf("Log output updated: %s -> %s", c.Log.Output, output)
			c.Log.Output = output
		}
	}

	// 缓存配置
	if cacheCfg, ok := config["cache"].(map[string]interface{}); ok {
		if enabled, ok := cacheCfg["enabled"].(bool); ok && enabled != c.Cache.Enabled {
			log.Printf("Cache enabled updated: %v -> %v", c.Cache.Enabled, enabled)
			c.Cache.Enabled = enabled
		}
		if ttl, ok := cacheCfg["ttl"].(string); ok && ttl != c.Cache.TTL {
			log.Printf("Cache TTL updated: %s -> %s", c.Cache.TTL, ttl)
			c.Cache.TTL = ttl
		}
	}

	log.Println("=== ServerConfig Reload Complete ===")
	return nil
}

// SimpleHTTPServer 简单的HTTP服务示例，演示配置热更新
type SimpleHTTPServer struct {
	config        *ServerConfig
	configManager *ConfigManager
}

// NewSimpleHTTPServer 创建HTTP服务
func NewSimpleHTTPServer(configManager *ConfigManager, serverConfig *ServerConfig) *SimpleHTTPServer {
	return &SimpleHTTPServer{
		config:        serverConfig,
		configManager: configManager,
	}
}

// Start 启动服务
func (s *SimpleHTTPServer) Start() error {
	mux := http.NewServeMux()

	// 获取配置信息的端点
	mux.HandleFunc("/config", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		config := s.configManager.GetCurrentConfig()
		fmt.Fprintf(w, "Current Config:\n")
		for k, v := range config {
			fmt.Fprintf(w, "%s: %v\n", k, v)
		}
	})

	// 服务信息端点
	mux.HandleFunc("/info", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprintf(w, "Service: %s\n", s.config.ServiceName)
		fmt.Fprintf(w, "Version: %s\n", s.config.Version)
		fmt.Fprintf(w, "Port: %d\n", s.config.Port)
		fmt.Fprintf(w, "Debug: %v\n", s.config.Debug)
		fmt.Fprintf(w, "Database: %s@%s:%d\n", s.config.Database.Username, s.config.Database.Host, s.config.Database.Port)
		fmt.Fprintf(w, "Features: %v\n", s.config.Features)
	})

	// 健康检查端点
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "OK")
	})

	addr := fmt.Sprintf(":%d", s.config.Port)
	log.Printf("HTTP Server starting on %s", addr)

	return http.ListenAndServe(addr, mux)
}

// RealWorldExample 真实场景示例
func RealWorldExample(configFilePath string) error {
	log.Println("=== Starting Real World Example ===")

	// 创建文件监控器
	watcherConfig := &FileWatcherConfig{
		FilePath:       configFilePath,
		FileType:       "json",
		WatchInterval:  2 * time.Second,
		EnableFsNotify: true,
	}

	watcher, err := NewFileConfigWatcher(watcherConfig)
	if err != nil {
		return fmt.Errorf("failed to create config watcher: %w", err)
	}

	// 创建配置管理器
	configManager := NewConfigManager(watcher)

	// 创建服务器配置
	serverConfig := &ServerConfig{
		ServiceName: "default-service",
		Version:     "0.0.0",
		Port:        8080,
		Debug:       false,
		Timeout:     30 * time.Second,
		Features:    []string{},
		Database: DatabaseConfig{
			Driver:   "mysql",
			Host:     "localhost",
			Port:     3306,
			Username: "root",
			Password: "password",
			Database: "app_db",
		},
		Log: LogConfig{
			Level:  "info",
			Format: "json",
			Output: "console",
		},
		Cache: CacheConfig{
			Enabled: false,
			TTL:     "1h",
		},
	}

	// 添加配置重载器
	configManager.AddReloader(serverConfig)

	// 启动配置管理器
	ctx := context.Background()
	if err := configManager.Start(ctx); err != nil {
		return fmt.Errorf("failed to start config manager: %w", err)
	}

	// 添加配置验证器
	validator := func(config map[string]interface{}) error {
		if port, ok := config["port"].(float64); ok {
			if port < 1 || port > 65535 {
				return fmt.Errorf("invalid port: %v, must be between 1 and 65535", port)
			}
		}
		return nil
	}

	if err := configManager.ValidateConfig(validator); err != nil {
		log.Printf("Warning: %v", err)
	}

	log.Printf("Configuration loaded successfully")
	log.Printf("Service: %s v%s", serverConfig.ServiceName, serverConfig.Version)
	log.Printf("Database: %s@%s:%d", serverConfig.Database.Username, serverConfig.Database.Host, serverConfig.Database.Port)
	log.Printf("Features: %v", serverConfig.Features)
	log.Printf("Log level: %s", serverConfig.Log.Level)

	// 创建并启动HTTP服务（在实际应用中）
	// server := NewSimpleHTTPServer(configManager, serverConfig)
	// go server.Start()

	log.Println("Configuration hot reload is now active")
	log.Println("You can modify the config file and changes will be automatically loaded")

	// 监听配置变更
	configManager.AddReloader(&ConfigLogger{})

	defer configManager.Stop()

	// 运行示例（实际应用中这里会是无限循环服务）
	return nil
}

// ConfigLogger 配置日志记录器，用于演示回调
type ConfigLogger struct{}

// Reload 记录所有配置变更
func (cl *ConfigLogger) Reload(config map[string]interface{}) error {
	log.Printf("Configuration changed at %s", time.Now().Format(time.RFC3339))
	return nil
}
