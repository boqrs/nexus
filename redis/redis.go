package redis

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// Mode 表示Redis部署模式
type Mode string

const (
	ModeStandalone Mode = "standalone" // 单例模式
	ModeCluster    Mode = "cluster"    // 集群模式
)

// Config Redis配置
type Config struct {
	Mode     Mode     `json:"mode" yaml:"mode" mapstructure:"mode"`                // 部署模式：standalone 或 cluster
	Addrs    []string `json:"addrs" yaml:"addrs" mapstructure:"addrs"`             // Redis地址列表
	Password string   `json:"password" yaml:"password" mapstructure:"password"`    // 密码
	DB       int      `json:"db" yaml:"db" mapstructure:"db"`                      // 数据库编号（仅单例模式有效）
	PoolSize int      `json:"pool_size" yaml:"pool_size" mapstructure:"pool_size"` // 连接池大小
}

// Client Redis客户端接口，屏蔽单例和集群差异
type Client interface {
	// 基础操作
	Get(ctx context.Context, key string) *redis.StringCmd
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd
	Del(ctx context.Context, keys ...string) *redis.IntCmd
	Exists(ctx context.Context, keys ...string) *redis.IntCmd
	Expire(ctx context.Context, key string, expiration time.Duration) *redis.BoolCmd
	TTL(ctx context.Context, key string) *redis.DurationCmd

	// 哈希操作
	HGet(ctx context.Context, key, field string) *redis.StringCmd
	HSet(ctx context.Context, key string, values ...interface{}) *redis.IntCmd
	HGetAll(ctx context.Context, key string) *redis.MapStringStringCmd
	HDel(ctx context.Context, key string, fields ...string) *redis.IntCmd
	HExists(ctx context.Context, key, field string) *redis.BoolCmd

	// 列表操作
	LPush(ctx context.Context, key string, values ...interface{}) *redis.IntCmd
	RPush(ctx context.Context, key string, values ...interface{}) *redis.IntCmd
	LPop(ctx context.Context, key string) *redis.StringCmd
	RPop(ctx context.Context, key string) *redis.StringCmd
	LRange(ctx context.Context, key string, start, stop int64) *redis.StringSliceCmd
	LLen(ctx context.Context, key string) *redis.IntCmd

	// 集合操作
	SAdd(ctx context.Context, key string, members ...interface{}) *redis.IntCmd
	SRem(ctx context.Context, key string, members ...interface{}) *redis.IntCmd
	SMembers(ctx context.Context, key string) *redis.StringSliceCmd
	SIsMember(ctx context.Context, key string, member interface{}) *redis.BoolCmd
	SCard(ctx context.Context, key string) *redis.IntCmd

	// 有序集合操作
	ZAdd(ctx context.Context, key string, members ...redis.Z) *redis.IntCmd
	ZRem(ctx context.Context, key string, members ...interface{}) *redis.IntCmd
	ZRange(ctx context.Context, key string, start, stop int64) *redis.StringSliceCmd
	ZRangeWithScores(ctx context.Context, key string, start, stop int64) *redis.ZSliceCmd
	ZScore(ctx context.Context, key string, member string) *redis.FloatCmd
	ZCard(ctx context.Context, key string) *redis.IntCmd

	// 发布订阅
	Publish(ctx context.Context, channel string, message interface{}) *redis.IntCmd
	Subscribe(ctx context.Context, channels ...string) *redis.PubSub

	// 事务
	TxPipeline() redis.Pipeliner
	TxPipelined(ctx context.Context, fn func(redis.Pipeliner) error) ([]redis.Cmder, error)

	// 连接管理
	Ping(ctx context.Context) *redis.StatusCmd
	Close() error
}

// standaloneClient 单例Redis客户端
type standaloneClient struct {
	client *redis.Client
}

func newStandaloneClient(cfg *Config) (Client, error) {
	if len(cfg.Addrs) == 0 {
		cfg.Addrs = []string{"localhost:6379"}
	}

	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addrs[0],
		Password: cfg.Password,
		DB:       cfg.DB,
		PoolSize: cfg.PoolSize,
	})

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	return &standaloneClient{client: client}, nil
}

func (c *standaloneClient) Get(ctx context.Context, key string) *redis.StringCmd {
	return c.client.Get(ctx, key)
}

func (c *standaloneClient) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd {
	return c.client.Set(ctx, key, value, expiration)
}

func (c *standaloneClient) Del(ctx context.Context, keys ...string) *redis.IntCmd {
	return c.client.Del(ctx, keys...)
}

func (c *standaloneClient) Exists(ctx context.Context, keys ...string) *redis.IntCmd {
	return c.client.Exists(ctx, keys...)
}

func (c *standaloneClient) Expire(ctx context.Context, key string, expiration time.Duration) *redis.BoolCmd {
	return c.client.Expire(ctx, key, expiration)
}

func (c *standaloneClient) TTL(ctx context.Context, key string) *redis.DurationCmd {
	return c.client.TTL(ctx, key)
}

func (c *standaloneClient) HGet(ctx context.Context, key, field string) *redis.StringCmd {
	return c.client.HGet(ctx, key, field)
}

func (c *standaloneClient) HSet(ctx context.Context, key string, values ...interface{}) *redis.IntCmd {
	return c.client.HSet(ctx, key, values...)
}

func (c *standaloneClient) HGetAll(ctx context.Context, key string) *redis.MapStringStringCmd {
	return c.client.HGetAll(ctx, key)
}

func (c *standaloneClient) HDel(ctx context.Context, key string, fields ...string) *redis.IntCmd {
	return c.client.HDel(ctx, key, fields...)
}

func (c *standaloneClient) HExists(ctx context.Context, key, field string) *redis.BoolCmd {
	return c.client.HExists(ctx, key, field)
}

func (c *standaloneClient) LPush(ctx context.Context, key string, values ...interface{}) *redis.IntCmd {
	return c.client.LPush(ctx, key, values...)
}

func (c *standaloneClient) RPush(ctx context.Context, key string, values ...interface{}) *redis.IntCmd {
	return c.client.RPush(ctx, key, values...)
}

func (c *standaloneClient) LPop(ctx context.Context, key string) *redis.StringCmd {
	return c.client.LPop(ctx, key)
}

func (c *standaloneClient) RPop(ctx context.Context, key string) *redis.StringCmd {
	return c.client.RPop(ctx, key)
}

func (c *standaloneClient) LRange(ctx context.Context, key string, start, stop int64) *redis.StringSliceCmd {
	return c.client.LRange(ctx, key, start, stop)
}

func (c *standaloneClient) LLen(ctx context.Context, key string) *redis.IntCmd {
	return c.client.LLen(ctx, key)
}

func (c *standaloneClient) SAdd(ctx context.Context, key string, members ...interface{}) *redis.IntCmd {
	return c.client.SAdd(ctx, key, members...)
}

func (c *standaloneClient) SRem(ctx context.Context, key string, members ...interface{}) *redis.IntCmd {
	return c.client.SRem(ctx, key, members...)
}

func (c *standaloneClient) SMembers(ctx context.Context, key string) *redis.StringSliceCmd {
	return c.client.SMembers(ctx, key)
}

func (c *standaloneClient) SIsMember(ctx context.Context, key string, member interface{}) *redis.BoolCmd {
	return c.client.SIsMember(ctx, key, member)
}

func (c *standaloneClient) SCard(ctx context.Context, key string) *redis.IntCmd {
	return c.client.SCard(ctx, key)
}

func (c *standaloneClient) ZAdd(ctx context.Context, key string, members ...redis.Z) *redis.IntCmd {
	return c.client.ZAdd(ctx, key, members...)
}

func (c *standaloneClient) ZRem(ctx context.Context, key string, members ...interface{}) *redis.IntCmd {
	return c.client.ZRem(ctx, key, members...)
}

func (c *standaloneClient) ZRange(ctx context.Context, key string, start, stop int64) *redis.StringSliceCmd {
	return c.client.ZRange(ctx, key, start, stop)
}

func (c *standaloneClient) ZRangeWithScores(ctx context.Context, key string, start, stop int64) *redis.ZSliceCmd {
	return c.client.ZRangeWithScores(ctx, key, start, stop)
}

func (c *standaloneClient) ZScore(ctx context.Context, key string, member string) *redis.FloatCmd {
	return c.client.ZScore(ctx, key, member)
}

func (c *standaloneClient) ZCard(ctx context.Context, key string) *redis.IntCmd {
	return c.client.ZCard(ctx, key)
}

func (c *standaloneClient) Publish(ctx context.Context, channel string, message interface{}) *redis.IntCmd {
	return c.client.Publish(ctx, channel, message)
}

func (c *standaloneClient) Subscribe(ctx context.Context, channels ...string) *redis.PubSub {
	return c.client.Subscribe(ctx, channels...)
}

func (c *standaloneClient) TxPipeline() redis.Pipeliner {
	return c.client.TxPipeline()
}

func (c *standaloneClient) TxPipelined(ctx context.Context, fn func(redis.Pipeliner) error) ([]redis.Cmder, error) {
	return c.client.TxPipelined(ctx, fn)
}

func (c *standaloneClient) Ping(ctx context.Context) *redis.StatusCmd {
	return c.client.Ping(ctx)
}

func (c *standaloneClient) Close() error {
	return c.client.Close()
}

// clusterClient 集群Redis客户端
type clusterClient struct {
	client *redis.ClusterClient
}

func newClusterClient(cfg *Config) (Client, error) {
	if len(cfg.Addrs) == 0 {
		cfg.Addrs = []string{"localhost:6379"}
	}

	client := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs:    cfg.Addrs,
		Password: cfg.Password,
		PoolSize: cfg.PoolSize,
	})

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	return &clusterClient{client: client}, nil
}

func (c *clusterClient) Get(ctx context.Context, key string) *redis.StringCmd {
	return c.client.Get(ctx, key)
}

func (c *clusterClient) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd {
	return c.client.Set(ctx, key, value, expiration)
}

func (c *clusterClient) Del(ctx context.Context, keys ...string) *redis.IntCmd {
	return c.client.Del(ctx, keys...)
}

func (c *clusterClient) Exists(ctx context.Context, keys ...string) *redis.IntCmd {
	return c.client.Exists(ctx, keys...)
}

func (c *clusterClient) Expire(ctx context.Context, key string, expiration time.Duration) *redis.BoolCmd {
	return c.client.Expire(ctx, key, expiration)
}

func (c *clusterClient) TTL(ctx context.Context, key string) *redis.DurationCmd {
	return c.client.TTL(ctx, key)
}

func (c *clusterClient) HGet(ctx context.Context, key, field string) *redis.StringCmd {
	return c.client.HGet(ctx, key, field)
}

func (c *clusterClient) HSet(ctx context.Context, key string, values ...interface{}) *redis.IntCmd {
	return c.client.HSet(ctx, key, values...)
}

func (c *clusterClient) HGetAll(ctx context.Context, key string) *redis.MapStringStringCmd {
	return c.client.HGetAll(ctx, key)
}

func (c *clusterClient) HDel(ctx context.Context, key string, fields ...string) *redis.IntCmd {
	return c.client.HDel(ctx, key, fields...)
}

func (c *clusterClient) HExists(ctx context.Context, key, field string) *redis.BoolCmd {
	return c.client.HExists(ctx, key, field)
}

func (c *clusterClient) LPush(ctx context.Context, key string, values ...interface{}) *redis.IntCmd {
	return c.client.LPush(ctx, key, values...)
}

func (c *clusterClient) RPush(ctx context.Context, key string, values ...interface{}) *redis.IntCmd {
	return c.client.RPush(ctx, key, values...)
}

func (c *clusterClient) LPop(ctx context.Context, key string) *redis.StringCmd {
	return c.client.LPop(ctx, key)
}

func (c *clusterClient) RPop(ctx context.Context, key string) *redis.StringCmd {
	return c.client.RPop(ctx, key)
}

func (c *clusterClient) LRange(ctx context.Context, key string, start, stop int64) *redis.StringSliceCmd {
	return c.client.LRange(ctx, key, start, stop)
}

func (c *clusterClient) LLen(ctx context.Context, key string) *redis.IntCmd {
	return c.client.LLen(ctx, key)
}

func (c *clusterClient) SAdd(ctx context.Context, key string, members ...interface{}) *redis.IntCmd {
	return c.client.SAdd(ctx, key, members...)
}

func (c *clusterClient) SRem(ctx context.Context, key string, members ...interface{}) *redis.IntCmd {
	return c.client.SRem(ctx, key, members...)
}

func (c *clusterClient) SMembers(ctx context.Context, key string) *redis.StringSliceCmd {
	return c.client.SMembers(ctx, key)
}

func (c *clusterClient) SIsMember(ctx context.Context, key string, member interface{}) *redis.BoolCmd {
	return c.client.SIsMember(ctx, key, member)
}

func (c *clusterClient) SCard(ctx context.Context, key string) *redis.IntCmd {
	return c.client.SCard(ctx, key)
}

func (c *clusterClient) ZAdd(ctx context.Context, key string, members ...redis.Z) *redis.IntCmd {
	return c.client.ZAdd(ctx, key, members...)
}

func (c *clusterClient) ZRem(ctx context.Context, key string, members ...interface{}) *redis.IntCmd {
	return c.client.ZRem(ctx, key, members...)
}

func (c *clusterClient) ZRange(ctx context.Context, key string, start, stop int64) *redis.StringSliceCmd {
	return c.client.ZRange(ctx, key, start, stop)
}

func (c *clusterClient) ZRangeWithScores(ctx context.Context, key string, start, stop int64) *redis.ZSliceCmd {
	return c.client.ZRangeWithScores(ctx, key, start, stop)
}

func (c *clusterClient) ZScore(ctx context.Context, key string, member string) *redis.FloatCmd {
	return c.client.ZScore(ctx, key, member)
}

func (c *clusterClient) ZCard(ctx context.Context, key string) *redis.IntCmd {
	return c.client.ZCard(ctx, key)
}

func (c *clusterClient) Publish(ctx context.Context, channel string, message interface{}) *redis.IntCmd {
	return c.client.Publish(ctx, channel, message)
}

func (c *clusterClient) Subscribe(ctx context.Context, channels ...string) *redis.PubSub {
	return c.client.Subscribe(ctx, channels...)
}

func (c *clusterClient) TxPipeline() redis.Pipeliner {
	return c.client.TxPipeline()
}

func (c *clusterClient) TxPipelined(ctx context.Context, fn func(redis.Pipeliner) error) ([]redis.Cmder, error) {
	return c.client.TxPipelined(ctx, fn)
}

func (c *clusterClient) Ping(ctx context.Context) *redis.StatusCmd {
	return c.client.Ping(ctx)
}

func (c *clusterClient) Close() error {
	return c.client.Close()
}

// NewClient 根据配置创建Redis客户端，自动选择单例或集群模式
func NewClient(cfg *Config) (Client, error) {
	switch cfg.Mode {
	case ModeStandalone:
		return newStandaloneClient(cfg)
	case ModeCluster:
		return newClusterClient(cfg)
	default:
		// 默认使用单例模式
		return newStandaloneClient(cfg)
	}
}
