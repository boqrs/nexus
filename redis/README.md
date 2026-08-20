# Redis 模块

这个模块提供了一个统一的Redis客户端接口，屏蔽了单例和集群模式的差异。

## 特性

- **统一接口**：单例和集群模式使用相同的API
- **自动模式选择**：根据配置自动选择合适的客户端实现
- **完整的数据类型支持**：支持字符串、哈希、列表、集合、有序集合等
- **连接池管理**：内置连接池，支持连接复用
- **上下文支持**：支持Go context，用于超时控制和取消操作

## 配置

```go
type Config struct {
    Mode     Mode     `json:"mode" yaml:"mode"`         // 部署模式：standalone 或 cluster
    Addrs    []string `json:"addrs" yaml:"addrs"`       // Redis地址列表
    Password string   `json:"password" yaml:"password"` // 密码
    DB       int      `json:"db" yaml:"db"`             // 数据库编号（仅单例模式有效）
    PoolSize int      `json:"pool_size" yaml:"pool_size"` // 连接池大小
}
```

### 单例模式配置

```go
config := &redis.Config{
    Mode:     redis.ModeStandalone,
    Addrs:    []string{"localhost:6379"},
    Password: "your-password",
    DB:       0,
    PoolSize: 10,
}
```

### 集群模式配置

```go
config := &redis.Config{
    Mode:     redis.ModeCluster,
    Addrs:    []string{"redis-node1:6379", "redis-node2:6379", "redis-node3:6379"},
    Password: "your-password",
    PoolSize: 20,
}
```

## 使用方法

### 初始化客户端

```go
client, err := redis.NewClient(config)
if err != nil {
    log.Fatal("Failed to create Redis client", err)
}
defer client.Close()
```

### 基础操作

```go
ctx := context.Background()

// 设置键值对
err := client.Set(ctx, "key", "value", time.Minute*5).Err()

// 获取值
value, err := client.Get(ctx, "key").Result()

// 删除键
deleted, err := client.Del(ctx, "key").Result()

// 检查键是否存在
exists, err := client.Exists(ctx, "key").Result()
```

### 哈希操作

```go
// 设置哈希字段
err := client.HSet(ctx, "user:123", "name", "张三").Err()
err = client.HSet(ctx, "user:123", "email", "zhangsan@example.com").Err()

// 获取哈希字段
name, err := client.HGet(ctx, "user:123", "name").Result()

// 获取所有字段
user, err := client.HGetAll(ctx, "user:123").Result()
```

### 列表操作

```go
// 推入列表
err := client.LPush(ctx, "queue", "item1", "item2").Err()

// 从列表弹出
item, err := client.LPop(ctx, "queue").Result()

// 获取列表范围
items, err := client.LRange(ctx, "queue", 0, -1).Result()
```

### 集合操作

```go
// 添加成员
err := client.SAdd(ctx, "tags", "golang", "redis").Err()

// 检查成员是否存在
exists, err := client.SIsMember(ctx, "tags", "golang").Result()

// 获取所有成员
members, err := client.SMembers(ctx, "tags").Result()
```

### 有序集合操作

```go
// 添加成员（带分数）
err := client.ZAdd(ctx, "leaderboard",
    redis.Z{Score: 100, Member: "player1"},
    redis.Z{Score: 150, Member: "player2"}).Err()

// 获取排名
players, err := client.ZRange(ctx, "leaderboard", 0, -1).Result()

// 获取分数
score, err := client.ZScore(ctx, "leaderboard", "player1").Result()
```

### 发布订阅

```go
// 发布消息
err := client.Publish(ctx, "channel", "message").Err()

// 订阅频道
pubsub := client.Subscribe(ctx, "channel")
defer pubsub.Close()

for msg := range pubsub.Channel() {
    fmt.Println(msg.Payload)
}
```

### 事务

```go
// 管道事务
_, err := client.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
    pipe.Set(ctx, "key1", "value1", 0)
    pipe.Set(ctx, "key2", "value2", 0)
    pipe.Del(ctx, "key3")
    return nil
})
```

## 示例应用

在 `example/main.go` 中包含了完整的Redis使用示例，包括：

- 基础的GET/SET操作
- 哈希数据结构操作
- 列表操作
- 集合操作
- 用户缓存示例
- 订单处理状态跟踪

### 启动示例应用

```bash
# 确保Redis服务器运行在localhost:6379
go run ./example/main.go
```

### 测试API端点

```bash
# 设置字符串值
curl "http://localhost:8080/redis/set?key=test&value=hello"

# 获取字符串值
curl "http://localhost:8080/redis/get?key=test"

# 设置哈希字段
curl "http://localhost:8080/redis/hash/set?key=user:1&field=name&value=张三"

# 获取哈希字段
curl "http://localhost:8080/redis/hash/get?key=user:1&field=name"

# 推入列表
curl "http://localhost:8080/redis/list/push?key=tasks&value=task1"

# 获取列表内容
curl "http://localhost:8080/redis/list/range?key=tasks"

# 添加集合成员
curl "http://localhost:8080/redis/set/add?key=tags&member=golang"

# 获取集合成员
curl "http://localhost:8080/redis/set/members?key=tags"
```

## 测试

运行单元测试：

```bash
go test ./redis/
```

注意：测试需要一个运行中的Redis服务器。如果没有Redis服务器，测试会被跳过。

## 部署模式差异

### 单例模式 (Standalone)
- 使用单个Redis实例
- 支持多数据库（DB 0-15）
- 适合开发环境和小型应用

### 集群模式 (Cluster)
- 使用Redis Cluster
- 自动分片和故障转移
- 不支持多数据库（所有键都在DB 0）
- 适合高可用性和大规模应用

## 错误处理

所有操作都返回 `*redis.Cmd` 类型，可以通过以下方式处理错误：

```go
result := client.Get(ctx, "key")
if result.Err() != nil {
    // 处理错误
    log.Printf("Redis error: %v", result.Err())
    return
}

value := result.Val()
```

## 连接管理

- 客户端会自动管理连接池
- 使用 `defer client.Close()` 确保资源释放
- 支持上下文超时控制
- 自动重连机制