# ConfigV2 - Viper文件配置监控器

这是一个基于[Viper](https://github.com/spf13/viper)的配置文件监控和热更新解决方案，用于替代原来基于Kubernetes ConfigMap的config模块。完全兼容原接口，支持JSON、YAML、TOML等多种配置文件格式，无需Kubernetes依赖。

## 特性

- 🚀 **文件监控**: 支持监控本地配置文件的变化
- 🔄 **热更新**: 配置变更时自动重新加载，无需重启应用
- 📁 **多格式支持**: 支持JSON、YAML、TOML等多种配置文件格式
- ⚡ **高效监控**: 支持两种监控方式：
  - **FSNotify**: 基于文件系统事件的实时监控（默认，更高效，毫秒级延迟）
  - **轮询**: 定时轮询文件修改时间（备选方案，通用性强）
- 🔗 **完全兼容**: 保持与原config模块相同的接口，平滑迁移
- 🛡️ **并发安全**: 使用RWMutex保护配置访问
- 📊 **易用API**: 提供便捷的配置值获取方法（GetString、GetInt等）
- ✅ **配置验证**: 内置配置验证机制
- 📦 **轻量依赖**: 仅200KB，远小于K8s客户端依赖

## 架构设计

### 核心组件

#### 1. FileConfigWatcher - 文件配置监控器

监控本地配置文件的变化，使用Viper库加载和解析配置。

```go
type FileWatcher struct {
    viper           *viper.Viper              // Viper实例
    filePath        string                    // 配置文件路径
    fileType        string                    // 文件类型 (json, yaml, toml等)
    currentConfig   map[string]interface{}    // 当前配置
    callbacks       []ConfigChangeCallback     // 变更回调
    stopCh          chan struct{}             // 停止信号
    running         bool                      // 运行状态
    watchInterval   time.Duration            // 轮询间隔
    enableFsNotify  bool                      // 是否使用fsnotify
}
```

**主要方法**：

- `NewFileConfigWatcher()` - 创建监控器
- `Start()` - 启动监控
- `Stop()` - 停止监控
- `AddCallback()` - 添加变更回调
- `GetCurrentConfig()` - 获取当前配置

#### 2. ConfigReloader - 配置重载器接口

定义配置重新加载的接口。

```go
type ConfigReloader interface {
    Reload(config map[string]interface{}) error
}
```

#### 3. ConfigManager - 配置管理器

管理多个配置重载器，协调配置更新。

```go
type ConfigManager struct {
    watcher   *FileConfigWatcher
    reloaders []ConfigReloader
}
```

**主要方法**：

- `NewConfigManager()` - 创建管理器
- `AddReloader()` - 添加重载器
- `Start()` - 启动管理器
- `Stop()` - 停止管理器
- `GetCurrentConfig()` - 获取当前配置
- `GetString()`, `GetInt()`, `GetBool()` 等便捷方法
- `ValidateConfig()` - 验证配置

## 使用方法

### 基础使用

#### 1. 创建配置文件

**config.json**:

```json
{
  "service_name": "my-service",
  "version": "1.0.0",
  "port": 8080,
  "debug": false,
  "timeout": "30s",
  "features": ["tracing", "metrics"],
  "database": {
    "driver": "mysql",
    "host": "localhost",
    "port": 3306,
    "database": "app_db"
  }
}
```

或 **config.yaml**:

```yaml
service_name: my-service
version: 1.0.0
port: 8080
debug: false
timeout: 30s
features:
  - tracing
  - metrics
database:
  driver: mysql
  host: localhost
  port: 3306
  database: app_db
```

#### 2. 实现配置重载器

```go
type AppConfig struct {
    ServiceName string
    Port        int
    Debug       bool
}

// 实现ConfigReloader接口
func (c *AppConfig) Reload(config map[string]interface{}) error {
    if name, ok := config["service_name"].(string); ok {
        c.ServiceName = name
        log.Printf("Updated service name: %s", name)
    }

    if port, ok := config["port"].(float64); ok {
        c.Port = int(port)
        log.Printf("Updated port: %d", c.Port)
    }

    if debug, ok := config["debug"].(bool); ok {
        c.Debug = debug
        log.Printf("Updated debug: %v", debug)
    }

    return nil
}
```

#### 3. 使用配置管理器

```go
package main

import (
    "context"
    "log"
    configV2 "codeup.aliyun.com/65b21d33076e069afe3d3253/basice/comm/configV2"
)

func main() {
    // 创建文件监控器配置
    watcherConfig := &configV2.FileWatcherConfig{
        FilePath:       "config.json",       // 配置文件路径
        FileType:       "json",              // 文件类型
        WatchInterval:  1 * time.Second,     // 轮询间隔（仅轮询模式）
        EnableFsNotify: true,                // 使用fsnotify
    }

    // 创建监控器
    watcher, err := configV2.NewFileConfigWatcher(watcherConfig)
    if err != nil {
        log.Fatalf("Failed to create config watcher: %v", err)
    }

    // 创建配置管理器
    configManager := configV2.NewConfigManager(watcher)

    // 创建应用配置
    appConfig := &AppConfig{}

    // 添加重载器
    configManager.AddReloader(appConfig)

    // 启动管理器
    ctx := context.Background()
    if err := configManager.Start(ctx); err != nil {
        log.Fatalf("Failed to start config manager: %v", err)
    }

    // 现在可以修改config.json文件，应用会自动重新加载配置

    // 优雅关闭
    defer configManager.Stop()
}
```

### 高级用法

#### 1. 使用便捷方法获取配置值

```go
// 获取简单配置值
serviceName := configManager.GetString("service_name")
port := configManager.GetInt("port")
debug := configManager.GetBool("debug")
features := configManager.GetStringSlice("features")

// 获取子配置对象
dbConfig := configManager.GetSubConfig("database")
```

#### 2. 添加配置验证

```go
// 定义验证函数
validator := func(config map[string]interface{}) error {
    if port, ok := config["port"].(float64); ok {
        if port < 1 || port > 65535 {
            return fmt.Errorf("invalid port: %v", port)
        }
    }
    return nil
}

// 验证配置
if err := configManager.ValidateConfig(validator); err != nil {
    log.Fatalf("Config validation failed: %v", err)
}
```

#### 3. 添加多个重载器

```go
configManager.AddReloader(appConfig)
configManager.AddReloader(databaseConfig)
configManager.AddReloader(cacheConfig)

// 所有重载器都会在配置变更时被调用
```

#### 4. 自定义变更回调

```go
watcherConfig := &configV2.FileWatcherConfig{
    FilePath: "config.json",
    FileType: "json",
    OnConfigChange: func(oldConfig, newConfig map[string]interface{}) error {
        log.Printf("Config changed from %v to %v", oldConfig, newConfig)
        // 执行其他操作
        return nil
    },
}

watcher, _ := configV2.NewFileConfigWatcher(watcherConfig)

// 后续可以继续添加更多回调
watcher.AddCallback(func(oldConfig, newConfig map[string]interface{}) error {
    // 第二个回调函数
    return nil
})
```

## 监控方式对比

### FSNotify 方式（推荐）

- **优点**: 文件变化时立即检测，性能好，资源占用少
- **缺点**: 需要依赖fsnotify库，某些文件系统可能不支持
- **适用场景**: 生产环境，文件变化不频繁

```go
FileWatcherConfig{
    EnableFsNotify: true,  // 使用fsnotify
}
```

### 轮询方式（备选）

- **优点**: 通用性强，不依赖特殊库
- **缺点**: 需要定时检查，有延迟，资源占用相对多
- **适用场景**: 文件系统不支持fsnotify的环境

```go
FileWatcherConfig{
    EnableFsNotify: false,
    WatchInterval:  1 * time.Second,  // 轮询间隔
}
```

## 从config迁移到configV2

### 迁移步骤

1. **替换导入**

   ```go
   // 旧
   import "codeup.aliyun.com/65b21d33076e069afe3d3253/basice/comm/config"

   // 新
   import configV2 "codeup.aliyun.com/65b21d33076e069afe3d3253/basice/comm/configV2"
   ```

2. **替换监控器创建**

   ```go
   // 旧
   watcher, err := config.NewConfigWatcher(&config.ConfigWatcherConfig{
       Namespace:     "default",
       ConfigMapName: "app-config",
       ConfigKey:     "config.json",
   })

   // 新
   watcher, err := configV2.NewFileConfigWatcher(&configV2.FileWatcherConfig{
       FilePath: "config.json",
       FileType: "json",
   })
   ```

3. **保持重载器和管理器接口不变**
   ```go
   // 重载器接口完全相同，无需修改
   configManager := configV2.NewConfigManager(watcher)
   configManager.AddReloader(appConfig)
   configManager.Start(ctx)
   ```

## 最佳实践

### 1. 配置文件位置

```
project/
├── config/
│   ├── config.json          # 默认配置
│   ├── config.prod.json     # 生产环境配置
│   └── config.dev.json      # 开发环境配置
└── main.go
```

### 2. 环境变量支持

```go
configPath := os.Getenv("CONFIG_PATH")
if configPath == "" {
    configPath = "config.json"
}

watcherConfig := &configV2.FileWatcherConfig{
    FilePath: configPath,
}
```

### 3. 错误处理

```go
if err := configManager.Start(ctx); err != nil {
    log.Fatalf("Failed to start config manager: %v", err)
}

// 配置验证失败时的处理
if err := configManager.ValidateConfig(validator); err != nil {
    log.Printf("Config validation warning: %v", err)
    // 可以选择继续运行或退出
}
```

### 4. 优雅关闭

```go
// 使用context控制生命周期
ctx, cancel := context.WithCancel(context.Background())

go func() {
    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
    <-sigCh
    cancel()
    configManager.Stop()
}()

if err := configManager.Start(ctx); err != nil {
    log.Fatalf("Failed to start: %v", err)
}
```

## 依赖

- `github.com/spf13/viper` - 配置文件解析
- `github.com/fsnotify/fsnotify` - 文件系统监控（可选，用于fsnotify方式）

## 常见问题

### Q: 为什么选择Viper而不是其他配置库？

A: Viper是Go生态中最流行的配置管理库，支持多种格式、环境变量、远程配置等。

### Q: 配置变更后多久会被检测到？

A:

- FSNotify方式：几毫秒内
- 轮询方式：1秒内（取决于WatchInterval）

### Q: 支持嵌套配置吗？

A: 支持，GetSubConfig方法可以获取嵌套配置对象。

### Q: 如果配置验证失败怎么办？

A: 可以在ValidateConfig返回错误，应用可以选择拒绝配置更新或记录警告。

## 性能考虑

1. **内存占用**: 配置被完整加载到内存，对于大型配置文件，建议定期检查
2. **文件I/O**: FSNotify方式的I/O操作最少
3. **CPU占用**: 轮询方式会增加CPU占用，建议间隔不要太短

## 安全性

1. 配置文件应该受到适当的文件权限保护
2. 敏感信息（如密码）可以通过环境变量注入
3. 建议对配置变更进行审计日志记录

---

## 快速开始指南

### 5分钟上手

#### 1. 准备配置文件

创建 `config.json`:

```json
{
  "service_name": "my-app",
  "port": 8080,
  "debug": false
}
```

#### 2. 定义配置结构

```go
package main

type AppConfig struct {
    ServiceName string
    Port        int
    Debug       bool
}

// 实现ConfigReloader接口
func (c *AppConfig) Reload(config map[string]interface{}) error {
    if name, ok := config["service_name"].(string); ok {
        c.ServiceName = name
    }
    if port, ok := config["port"].(float64); ok {
        c.Port = int(port)
    }
    if debug, ok := config["debug"].(bool); ok {
        c.Debug = debug
    }
    return nil
}
```

#### 3. 使用配置管理器

```go
package main

import (
    "context"
    "log"
    configV2 "your-module/configV2"
)

func main() {
    // 创建监控器
    watcher, err := configV2.NewFileConfigWatcher(&configV2.FileWatcherConfig{
        FilePath: "config.json",
        FileType: "json",
    })
    if err != nil {
        log.Fatal(err)
    }

    // 创建管理器
    manager := configV2.NewConfigManager(watcher)
    config := &AppConfig{}

    // 添加重载器并启动
    manager.AddReloader(config)
    if err := manager.Start(context.Background()); err != nil {
        log.Fatal(err)
    }

    // 应用会自动监控config.json的变化
    log.Printf("App: %s on port %d (debug=%v)\n",
        config.ServiceName, config.Port, config.Debug)

    // 优雅关闭
    defer manager.Stop()
}
```

#### 4. 测试热更新

在另一个终端修改 `config.json`:

```bash
# 改变port为9000
sed -i 's/"port": 8080/"port": 9000/g' config.json
# 应用会自动检测并重新加载配置
```

---

## 迁移指南：从config (K8s ConfigMap) 到 configV2 (Viper)

### 为什么迁移？

| 方面     | config (K8s) | configV2 (Viper) |
| -------- | ------------ | ---------------- |
| 依赖大小 | 数十MB       | 200KB            |
| K8s依赖  | 必需         | ❌ 不需要        |
| 本地开发 | 困难         | ✅ 容易          |
| 文件格式 | 仅JSON       | JSON/YAML/TOML   |
| 检测延迟 | 秒级         | **毫秒级**       |
| 接口兼容 | -            | **100%兼容**     |

### 迁移步骤

#### 步骤1: 准备配置文件

从K8s ConfigMap转换为本地文件：

**原来的K8s ConfigMap**:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: app-config
data:
  config.json: |
    {
      "service_name": "my-service",
      "port": 8080
    }
```

**转换为本地文件** (config.json):

```json
{
  "service_name": "my-service",
  "port": 8080
}
```

#### 步骤2: 更新导入

```go
// 旧
import "your-module/config"

// 新
import configV2 "your-module/configV2"
```

#### 步骤3: 更新初始化代码

**旧代码**:

```go
watcherConfig := &config.ConfigWatcherConfig{
    Namespace:     "default",
    ConfigMapName: "app-config",
    ConfigKey:     "config.json",
}
watcher, err := config.NewConfigWatcher(watcherConfig)
```

**新代码**:

```go
watcherConfig := &configV2.FileWatcherConfig{
    FilePath: "config.json",
    FileType: "json",
}
watcher, err := configV2.NewFileConfigWatcher(watcherConfig)
```

#### 步骤4: 配置重载器（无需改动）

配置重载器和管理器的接口完全相同，无需修改：

```go
// 这段代码在两个版本中完全相同
configManager := configV2.NewConfigManager(watcher)
appConfig := &MyAppConfig{}
configManager.AddReloader(appConfig)
if err := configManager.Start(ctx); err != nil {
    log.Fatal(err)
}
```

### 新增功能

#### 1. 便捷的值获取方法

```go
// 无需手动类型转换
serviceName := configManager.GetString("service_name")
port := configManager.GetInt("port")
debug := configManager.GetBool("debug")
features := configManager.GetStringSlice("features")
dbConfig := configManager.GetSubConfig("database")
```

#### 2. 配置验证

```go
validator := func(config map[string]interface{}) error {
    if port, ok := config["port"].(float64); ok {
        if port < 1 || port > 65535 {
            return fmt.Errorf("invalid port")
        }
    }
    return nil
}
configManager.ValidateConfig(validator)
```

#### 3. 多种文件格式支持

```go
// JSON
&configV2.FileWatcherConfig{FilePath: "config.json", FileType: "json"}

// YAML
&configV2.FileWatcherConfig{FilePath: "config.yaml", FileType: "yaml"}

// TOML
&configV2.FileWatcherConfig{FilePath: "config.toml", FileType: "toml"}
```

#### 4. 灵活的监控方式

```go
// 使用FSNotify（高效，默认）
&configV2.FileWatcherConfig{EnableFsNotify: true}

// 使用轮询（备选，通用）
&configV2.FileWatcherConfig{
    EnableFsNotify: false,
    WatchInterval:  1 * time.Second,
}
```

---

## 实现细节

### 核心设计

#### FileConfigWatcher（文件配置监控器）

**功能**:

- 使用Viper库加载和解析配置文件
- 监控文件变化（FSNotify或轮询）
- 管理回调函数列表
- 保证并发安全访问

**工作流程**:

1. 初始化时加载配置文件
2. 启动时注册fsnotify或启动轮询
3. 检测到变化时重新加载配置
4. 触发所有注册的回调函数

#### ConfigManager（配置管理器）

**功能**:

- 协调FileConfigWatcher和ConfigReloader
- 管理多个配置重载器
- 提供便捷的值获取API
- 支持配置验证

#### ConfigReloader（配置重载器接口）

```go
type ConfigReloader interface {
    Reload(config map[string]interface{}) error
}
```

### 并发安全

使用`sync.RWMutex`保护配置访问：

```go
// 读操作（无锁争用）
func (w *FileConfigWatcher) GetCurrentConfig() map[string]interface{} {
    w.mu.RLock()
    defer w.mu.RUnlock()
    // 返回配置副本
}

// 写操作（配置更新时）
w.mu.Lock()
w.currentConfig = newConfig
w.mu.Unlock()
```

这确保了：

- 读操作不阻塞其他读操作
- 写操作排他执行
- 避免读到半更新的状态

### 类型转换说明

配置从文件解析后为`map[string]interface{}`，各类型的转换规则：

| 文件类型 | JSON                   | YAML                   | 说明                     |
| -------- | ---------------------- | ---------------------- | ------------------------ |
| 数字     | float64                | float64                | JSON/YAML都解析为float64 |
| 字符串   | string                 | string                 | 保持字符串类型           |
| 布尔     | bool                   | bool                   | 保持布尔类型             |
| 数组     | []interface{}          | []interface{}          | 保持接口切片             |
| 对象     | map[string]interface{} | map[string]interface{} | 保持接口映射             |

### 监控方式对比

#### FSNotify 方式（推荐）

使用操作系统提供的文件系统事件，实时监控文件变化。

**优点**:

- 实时检测（毫秒级延迟）
- 低CPU占用
- 无轮询开销

**缺点**:

- 依赖fsnotify库
- 某些文件系统可能不支持

**配置**:

```go
FileWatcherConfig{
    EnableFsNotify: true,
}
```

#### 轮询方式（备选）

定时检查文件修改时间，兼容性好。

**优点**:

- 无额外依赖
- 通用性强

**缺点**:

- 有延迟（取决于WatchInterval）
- CPU占用相对高

**配置**:

```go
FileWatcherConfig{
    EnableFsNotify: false,
    WatchInterval:  1 * time.Second,
}
```

### 性能指标

| 指标     | FSNotify | 轮询     | 说明           |
| -------- | -------- | -------- | -------------- |
| 检测延迟 | 1-10ms   | 0-1000ms | FSNotify更快   |
| CPU占用  | 极低     | 轻微持续 | 轮询消耗更多   |
| 内存占用 | ~200KB   | ~200KB   | 相同           |
| 文件I/O  | 最少     | 定期检查 | FSNotify更高效 |

---

## 依赖说明

### 核心依赖

#### 必需

```
github.com/spf13/viper v1.15.0+
```

**用途**: 配置文件加载和解析

**功能**:

- 支持多种文件格式（JSON、YAML、TOML等）
- 自动类型转换
- 环境变量绑定（可选）

#### 可选

```
github.com/fsnotify/fsnotify v1.6.0+
```

**用途**: FSNotify方式文件监控（可选）

**说明**:

- 如果启用`EnableFsNotify: true`，需要此依赖
- 如果使用轮询方式`EnableFsNotify: false`，不需要此依赖
- Viper已经包含fsnotify作为传递依赖

### 版本要求

- **Go版本**: 推荐 Go 1.16+，最低 Go 1.11+
- **Viper版本**: 推荐 v1.15.0+

### 依赖大小

| 库       | 大小       | 说明                 |
| -------- | ---------- | -------------------- |
| viper    | ~100KB     | 配置框架             |
| fsnotify | ~20KB      | 文件监控（自动依赖） |
| **总计** | **~200KB** | **极其轻量**         |

相比原config模块的K8s依赖（数十MB），配置库的依赖极其轻量。

---

## 文件说明

- **file_watcher.go** - 文件配置监控器实现，支持FSNotify和轮询两种方式
- **config_manager.go** - 配置管理器实现和便捷API
- **example.go** - 基础使用示例和高级YAML示例
- **server_example.go** - 真实应用场景示例（HTTP服务）
- **file_watcher_test.go** - 完整的单元测试
- **config.example.json** - JSON配置文件示例
- **config.example.yaml** - YAML配置文件示例

---

## 常见问题

### Q: 如何在不同环境使用不同配置？

使用环境变量指定配置文件路径：

```go
configPath := os.Getenv("CONFIG_PATH")
if configPath == "" {
    configPath = "config.json"
}

watcherConfig := &configV2.FileWatcherConfig{
    FilePath: configPath,
}
```

### Q: 如何处理敏感信息（密码等）？

1. 使用环境变量注入敏感信息
2. 合并配置文件和环境变量
3. 使用Viper的BindEnv功能

```go
// 从环境变量补充敏感信息
dbPassword := os.Getenv("DB_PASSWORD")
if dbPassword != "" {
    config["database"].(map[string]interface{})["password"] = dbPassword
}
```

### Q: 如何处理配置校验失败？

使用ValidateConfig方法：

```go
if err := configManager.ValidateConfig(validator); err != nil {
    log.Printf("Config validation failed: %v", err)
    // 选择: 拒绝更新 或 记录警告继续
}
```

### Q: 监控文件的修改延迟是多少？

- FSNotify: 几毫秒
- 轮询: 取决于WatchInterval (推荐1秒)

### Q: 可以同时监控多个配置文件吗？

可以，创建多个FileConfigWatcher实例：

```go
watcher1 := configV2.NewFileConfigWatcher(&configV2.FileWatcherConfig{
    FilePath: "config.json",
})
watcher2 := configV2.NewFileConfigWatcher(&configV2.FileWatcherConfig{
    FilePath: "secrets.yaml",
})

manager1 := configV2.NewConfigManager(watcher1)
manager2 := configV2.NewConfigManager(watcher2)
```

### Q: 为什么配置没有更新？

检查以下几点：

1. 文件路径是否正确（相对或绝对路径）
2. 文件格式是否有效JSON/YAML/TOML
3. 是否实现了Reload方法
4. 文件是否有读权限

### Q: FSNotify不可用怎么办？

设置 `EnableFsNotify: false` 自动切换为轮询方式。

---

## 总结

configV2相比原config（K8s ConfigMap方案）的优势：

**✅ 优势**:

- 无需K8s依赖，更轻量
- 支持多种配置文件格式
- 检测速度更快（FSNotify毫秒级）
- 便捷的值获取API
- 内置配置验证
- 易于本地开发

**❌ 劣势**:

- 不支持集中式K8s配置管理
- 需要单独部署配置文件到各节点

**适用场景**:

- **configV2**: 本地开发、单机部署、非K8s环境、轻量级应用
- **原config**: K8s集群部署、需要集中配置管理、复杂多集群场景

两个方案的核心配置管理接口保持一致，可以平滑迁移。
