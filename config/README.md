# ConfigMap 配置监控器

这个模块提供了实时监控Kubernetes ConfigMap变化的功能，支持配置的动态更新。

## 功能特性

- 🚀 **实时监控**: 使用Kubernetes Watch API实时监控ConfigMap变化
- 🔄 **热更新**: 支持配置的热重载，无需重启应用
- 🛡️ **容错性**: 网络异常时自动重连，ConfigMap删除时记录警告
- 🎯 **类型安全**: 支持结构化配置和回调函数
- 📊 **可观测性**: 集成日志记录，方便调试和监控

## 使用方法

### 1. 创建ConfigMap

首先在Kubernetes中创建ConfigMap：

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: app-config
  namespace: default
data:
  config.json: |
    {
      "service_name": "my-service",
      "version": "1.0.0",
      "database": {
        "driver": "mysql",
        "host": "mysql-service",
        "port": 3306,
        "username": "app",
        "database": "app_db"
      },
      "timeout": "30s",
      "features": ["tracing", "metrics"]
    }
```

### 2. 初始化监控器

```go
package main

import (
    "context"
    "codeup.aliyun.com/65b21d33076e069afe3d3253/basice/comm/config"
    "codeup.aliyun.com/65b21d33076e069afe3d3253/basice/comm/log"
)

func main() {
    // 初始化日志
    logConfig := &log.Config{
        Level:   "info",
        Format:  "json",
        Output:  "console",
    }
    log.Init(logConfig)

    // 创建监控器配置
    watcherConfig := &config.ConfigWatcherConfig{
        Namespace:     "default",
        ConfigMapName: "app-config",
        ConfigKey:     "config.json",
        // KubeConfig: "/path/to/kubeconfig", // 可选：外部kubeconfig
    }

    // 创建监控器
    watcher, err := config.NewConfigWatcher(watcherConfig)
    if err != nil {
        log.Log().Fatal("Failed to create config watcher", err)
    }

    // 创建配置管理器
    configManager := config.NewConfigManager(watcher)

    // 创建配置实例
    appConfig := &MyAppConfig{}

    // 添加配置重载器
    configManager.AddReloader(appConfig)

    // 启动监控
    ctx := context.Background()
    if err := configManager.Start(ctx); err != nil {
        log.Log().Fatal("Failed to start config manager", err)
    }

    // 应用运行...
}
```

### 3. 实现配置重载器

```go
type MyAppConfig struct {
    ServiceName string        `json:"service_name"`
    Version     string        `json:"version"`
    Timeout     time.Duration `json:"timeout"`
    Features    []string      `json:"features"`
}

// 实现ConfigReloader接口
func (c *MyAppConfig) Reload(config map[string]interface{}) error {
    log.Log().Info("Reloading configuration", "config", config)

    // 更新配置逻辑
    if serviceName, ok := config["service_name"].(string); ok {
        c.ServiceName = serviceName
    }

    if timeoutStr, ok := config["timeout"].(string); ok {
        if timeout, err := time.ParseDuration(timeoutStr); err == nil {
            c.Timeout = timeout
        }
    }

    // 可以在这里更新数据库连接、日志级别等

    return nil
}
```

## 配置选项

### ConfigWatcherConfig

| 字段 | 类型 | 描述 | 默认值 |
|------|------|------|--------|
| `Namespace` | `string` | ConfigMap所在命名空间 | `"default"` |
| `ConfigMapName` | `string` | ConfigMap名称 | 必需 |
| `ConfigKey` | `string` | 配置在ConfigMap中的key | 必需 |
| `KubeConfig` | `string` | kubeconfig文件路径 | 空(使用集群内配置) |
| `ResyncPeriod` | `time.Duration` | 重新同步周期 | `5m` |
| `RetryDelay` | `time.Duration` | 重试延迟 | `5s` |
| `OnConfigChange` | `ConfigChangeCallback` | 配置变更回调 | `nil` |

## 部署说明

### 集群内部署

当应用部署在Kubernetes集群内时，无需额外配置：

```go
watcherConfig := &config.ConfigWatcherConfig{
    Namespace:     "your-namespace",
    ConfigMapName: "your-configmap",
    ConfigKey:     "config.json",
}
```

### 外部访问

当应用部署在集群外时，需要指定kubeconfig路径：

```go
watcherConfig := &config.ConfigWatcherConfig{
    KubeConfig:    "/path/to/kubeconfig",
    Namespace:     "your-namespace",
    ConfigMapName: "your-configmap",
    ConfigKey:     "config.json",
}
```

## RBAC权限

确保ServiceAccount具有以下权限：

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: config-watcher
rules:
- apiGroups: [""]
  resources: ["configmaps"]
  verbs: ["get", "list", "watch"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: config-watcher-binding
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: config-watcher
subjects:
- kind: ServiceAccount
  name: your-service-account
  namespace: your-namespace
```

## 错误处理

监控器会自动处理以下错误情况：

- **网络异常**: 自动重连，指数退避重试
- **ConfigMap不存在**: 记录错误，继续监控
- **配置解析失败**: 记录错误，保留旧配置
- **回调函数失败**: 记录错误，继续处理其他回调

## 日志输出

监控器会输出以下日志：

```
INFO ConfigMap watcher started namespace=default configmap=app-config key=config.json
INFO ConfigMap updated namespace=default name=app-config key=config.json
ERROR ConfigMap watch error error=...
```

## 示例代码

查看 `example.go` 文件中的完整示例：

```go
// 运行示例
config.ExampleUsage()

// 打印ConfigMap YAML示例
config.PrintExampleConfigMap()
```

## 注意事项

1. **配置格式**: ConfigMap中的配置必须是有效的JSON格式
2. **权限控制**: 确保Pod的ServiceAccount有足够的权限访问ConfigMap
3. **配置验证**: 在Reload方法中添加配置验证逻辑
4. **线程安全**: Reload方法需要是线程安全的
5. **错误处理**: 妥善处理配置更新失败的情况