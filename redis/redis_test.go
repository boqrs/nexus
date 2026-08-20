package redis

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

// mockRedisServer 模拟Redis服务器（仅用于测试）
type mockRedisServer struct {
	data map[string]interface{}
}

func newMockRedisServer() *mockRedisServer {
	return &mockRedisServer{
		data: make(map[string]interface{}),
	}
}

func (m *mockRedisServer) get(key string) (string, bool) {
	val, exists := m.data[key]
	if !exists {
		return "", false
	}
	if str, ok := val.(string); ok {
		return str, true
	}
	return "", false
}

func (m *mockRedisServer) set(key string, value interface{}) {
	m.data[key] = value
}

func (m *mockRedisServer) del(key string) {
	delete(m.data, key)
}

// TestStandaloneClient 测试单例模式客户端
func TestStandaloneClient(t *testing.T) {
	// 注意：这个测试需要一个真实的Redis服务器运行在localhost:6379
	// 在实际测试环境中，你可能需要启动一个Redis容器或使用内存模拟

	cfg := &Config{
		Mode:     ModeStandalone,
		Addrs:    []string{"localhost:6379"},
		Password: "",
		DB:       0,
		PoolSize: 10,
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Skipf("Skipping test due to Redis server not available: %v", err)
		return
	}
	defer client.Close()

	ctx := context.Background()

	// 测试基础操作
	t.Run("BasicOperations", func(t *testing.T) {
		key := "test:key"
		value := "test_value"

		// Set
		err := client.Set(ctx, key, value, 0).Err()
		if err != nil {
			t.Fatalf("Set failed: %v", err)
		}

		// Get
		result, err := client.Get(ctx, key).Result()
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}
		if result != value {
			t.Errorf("Expected %s, got %s", value, result)
		}

		// Exists
		count, err := client.Exists(ctx, key).Result()
		if err != nil {
			t.Fatalf("Exists failed: %v", err)
		}
		if count != 1 {
			t.Errorf("Expected 1, got %d", count)
		}

		// Del
		deleted, err := client.Del(ctx, key).Result()
		if err != nil {
			t.Fatalf("Del failed: %v", err)
		}
		if deleted != 1 {
			t.Errorf("Expected 1, got %d", deleted)
		}
	})

	// 测试哈希操作
	t.Run("HashOperations", func(t *testing.T) {
		key := "test:hash"

		// HSet
		err := client.HSet(ctx, key, "field1", "value1", "field2", "value2").Err()
		if err != nil {
			t.Fatalf("HSet failed: %v", err)
		}

		// HGet
		result, err := client.HGet(ctx, key, "field1").Result()
		if err != nil {
			t.Fatalf("HGet failed: %v", err)
		}
		if result != "value1" {
			t.Errorf("Expected value1, got %s", result)
		}

		// HGetAll
		all, err := client.HGetAll(ctx, key).Result()
		if err != nil {
			t.Fatalf("HGetAll failed: %v", err)
		}
		if len(all) != 2 {
			t.Errorf("Expected 2 fields, got %d", len(all))
		}

		// 清理
		client.Del(ctx, key)
	})

	// 测试列表操作
	t.Run("ListOperations", func(t *testing.T) {
		key := "test:list"

		// LPush
		err := client.LPush(ctx, key, "item1", "item2").Err()
		if err != nil {
			t.Fatalf("LPush failed: %v", err)
		}

		// LRange
		items, err := client.LRange(ctx, key, 0, -1).Result()
		if err != nil {
			t.Fatalf("LRange failed: %v", err)
		}
		if len(items) != 2 {
			t.Errorf("Expected 2 items, got %d", len(items))
		}

		// LLen
		length, err := client.LLen(ctx, key).Result()
		if err != nil {
			t.Fatalf("LLen failed: %v", err)
		}
		if length != 2 {
			t.Errorf("Expected length 2, got %d", length)
		}

		// 清理
		client.Del(ctx, key)
	})

	// 测试集合操作
	t.Run("SetOperations", func(t *testing.T) {
		key := "test:set"

		// SAdd
		err := client.SAdd(ctx, key, "member1", "member2").Err()
		if err != nil {
			t.Fatalf("SAdd failed: %v", err)
		}

		// SIsMember
		isMember, err := client.SIsMember(ctx, key, "member1").Result()
		if err != nil {
			t.Fatalf("SIsMember failed: %v", err)
		}
		if !isMember {
			t.Error("Expected member1 to be in set")
		}

		// SMembers
		members, err := client.SMembers(ctx, key).Result()
		if err != nil {
			t.Fatalf("SMembers failed: %v", err)
		}
		if len(members) != 2 {
			t.Errorf("Expected 2 members, got %d", len(members))
		}

		// 清理
		client.Del(ctx, key)
	})

	// 测试有序集合操作
	t.Run("SortedSetOperations", func(t *testing.T) {
		key := "test:zset"

		// ZAdd
		err := client.ZAdd(ctx, key, redis.Z{Score: 1.0, Member: "member1"}, redis.Z{Score: 2.0, Member: "member2"}).Err()
		if err != nil {
			t.Fatalf("ZAdd failed: %v", err)
		}

		// ZScore
		score, err := client.ZScore(ctx, key, "member1").Result()
		if err != nil {
			t.Fatalf("ZScore failed: %v", err)
		}
		if score != 1.0 {
			t.Errorf("Expected score 1.0, got %f", score)
		}

		// ZRange
		members, err := client.ZRange(ctx, key, 0, -1).Result()
		if err != nil {
			t.Fatalf("ZRange failed: %v", err)
		}
		if len(members) != 2 {
			t.Errorf("Expected 2 members, got %d", len(members))
		}

		// 清理
		client.Del(ctx, key)
	})

	// 测试过期时间
	t.Run("Expiration", func(t *testing.T) {
		key := "test:expire"

		// Set with expiration
		err := client.Set(ctx, key, "value", time.Second).Err()
		if err != nil {
			t.Fatalf("Set with expiration failed: %v", err)
		}

		// Check TTL
		ttl, err := client.TTL(ctx, key).Result()
		if err != nil {
			t.Fatalf("TTL failed: %v", err)
		}
		if ttl <= 0 {
			t.Error("Expected positive TTL")
		}

		// Wait for expiration
		time.Sleep(time.Second * 2)

		// Check if key exists
		exists, err := client.Exists(ctx, key).Result()
		if err != nil {
			t.Fatalf("Exists after expiration failed: %v", err)
		}
		if exists != 0 {
			t.Error("Expected key to be expired")
		}
	})
}

// TestClusterClient 测试集群模式客户端
func TestClusterClient(t *testing.T) {
	// 注意：这个测试需要一个Redis集群运行
	// 在实际测试环境中，你可能需要启动Redis集群或使用内存模拟

	cfg := &Config{
		Mode:     ModeCluster,
		Addrs:    []string{"localhost:6379"}, // 集群中的节点地址
		Password: "",
		PoolSize: 10,
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Skipf("Skipping test due to Redis cluster not available: %v", err)
		return
	}
	defer client.Close()

	ctx := context.Background()

	// 基础连接测试
	t.Run("Ping", func(t *testing.T) {
		result, err := client.Ping(ctx).Result()
		if err != nil {
			t.Fatalf("Ping failed: %v", err)
		}
		if result != "PONG" {
			t.Errorf("Expected PONG, got %s", result)
		}
	})
}

// TestConfigValidation 测试配置验证
func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name   string
		config *Config
		valid  bool
	}{
		{
			name: "ValidStandalone",
			config: &Config{
				Mode:  ModeStandalone,
				Addrs: []string{"localhost:6379"},
			},
			valid: true,
		},
		{
			name: "ValidCluster",
			config: &Config{
				Mode:  ModeCluster,
				Addrs: []string{"localhost:6379", "localhost:6380"},
			},
			valid: true,
		},
		{
			name: "DefaultMode",
			config: &Config{
				Addrs: []string{"localhost:6379"},
			},
			valid: true,
		},
		{
			name: "EmptyAddrs",
			config: &Config{
				Mode: ModeStandalone,
			},
			valid: true, // 应该使用默认地址
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.config)
			if tt.valid {
				if err != nil {
					t.Skipf("Skipping test due to Redis server not available: %v", err)
				}
				if client != nil {
					client.Close()
				}
			} else {
				if err == nil {
					t.Error("Expected error for invalid config")
					client.Close()
				}
			}
		})
	}
}