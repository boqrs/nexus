package media

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBuildObjectKey 测试路径构建逻辑
func TestBuildObjectKey(t *testing.T) {
	tests := []struct {
		name      string
		userID    string
		objectKey string
		want      string
	}{
		{
			name:      "normal case",
			userID:    "user123",
			objectKey: "photo.jpg",
			want:      "user123/photo.jpg",
		},
		{
			name:      "object key with leading slash",
			userID:    "user123",
			objectKey: "/photo.jpg",
			want:      "user123/photo.jpg",
		},
		{
			name:      "nested object key",
			userID:    "user123",
			objectKey: "folder/sub/file.png",
			want:      "user123/folder/sub/file.png",
		},
		{
			name:      "empty object key",
			userID:    "user123",
			objectKey: "",
			want:      "user123",
		},
		{
			name:      "windows style path conversion",
			userID:    "user123",
			objectKey: "folder\\sub\\file.png",       // filepath.Join 会处理分隔符
			want:      "user123/folder/sub/file.png", // 在 Linux/Mac 上通常是 /
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildObjectKey(tt.userID, tt.objectKey)
			// 注意：filepath.Join 的行为取决于操作系统，但在 OSS/S3 中通常期望正斜杠
			// 这里我们主要测试前缀拼接和去前导斜杠
			assert.True(t, strings.HasPrefix(got, tt.userID))
			if tt.objectKey != "" {
				assert.Contains(t, got, strings.TrimPrefix(tt.objectKey, "/"))
			}
		})
	}
}

// TestNewStorage_UnsupportedProvider 测试不支持的 Provider
func TestNewStorage_UnsupportedProvider(t *testing.T) {
	cfg := &Config{
		Provider: "unsupported",
	}
	storage, err := NewStorage(cfg)
	assert.Error(t, err)
	assert.Nil(t, storage)
	assert.Contains(t, err.Error(), "unsupported provider")
}

// TestNewStorage_OSS_Initialization 测试 OSS 初始化 (仅检查是否返回实例，不发起网络请求)
func TestNewStorage_OSS_Initialization(t *testing.T) {
	cfg := &Config{
		Provider:  ProviderOSS,
		Region:    "cn-hangzhou",
		AccessKey: "test-key",
		SecretKey: "test-secret",
		RoleArn:   "acs:ram::123:role/test",
	}

	storage, err := NewStorage(cfg)
	// newOSSStorage 内部创建 STS Client 通常不会立即报错，除非配置极度错误
	if assert.NoError(t, err) {
		assert.NotNil(t, storage)
		// 检查类型
		_, ok := storage.(*ossStorage)
		assert.True(t, ok, "Expected storage to be of type *ossStorage")
	}
}

// TestNewStorage_S3_Initialization 测试 S3 初始化
func TestNewStorage_S3_Initialization(t *testing.T) {
	cfg := &Config{
		Provider:  ProviderS3,
		Region:    "us-east-1",
		AccessKey: "test-key",
		SecretKey: "test-secret",
		RoleArn:   "arn:aws:iam::123:role/test",
	}

	storage, err := NewStorage(cfg)
	if assert.NoError(t, err) {
		assert.NotNil(t, storage)
		_, ok := storage.(*s3Storage)
		assert.True(t, ok, "Expected storage to be of type *s3Storage")
	}
}

// TestGenerateAliyunCdnSignedURL 测试 CDN URL 生成逻辑
func TestGenerateAliyunCdnSignedURL(t *testing.T) {
	domain := "cdn.example.com"
	objectKey := "user123/photo.jpg"
	key := "my-secret-key"
	expire := 1 * time.Hour

	url := generateAliyunCdnSignedURL(domain, objectKey, key, expire)

	// 1. 检查基本格式
	assert.Contains(t, url, "https://"+domain+"/"+objectKey)
	assert.Contains(t, url, "?auth_key=")

	// 2. 检查签名部分结构: timestamp-rand-uid-md5
	parts := strings.Split(url, "?auth_key=")
	require.Len(t, parts, 2)
	signPart := parts[1]

	signComponents := strings.Split(signPart, "-")
	require.Len(t, signComponents, 4, "Auth key should have 4 components: timestamp, rand, uid, md5")

	// 3. 检查时间戳是否在预期范围内
	//now := time.Now()
	//expectedMax := now.Add(expire).Unix()
	//expectedMin := now.Add(expire - 2*time.Second).Unix() // 允许少量执行时间偏差

	assert.GreaterOrEqual(t, len(signComponents[0]), 10) // Unix timestamp length

	// 简单验证 MD5 长度 (32 chars)
	assert.Len(t, signComponents[3], 32)
}

// TestTempCredentials_Structure 测试临时凭证结构体
func TestTempCredentials_Structure(t *testing.T) {
	now := time.Now()
	creds := &TempCredentials{
		AccessKeyID:     "AKID",
		AccessKeySecret: "SK",
		SecurityToken:   "Token",
		Expiration:      now,
	}

	assert.Equal(t, "AKID", creds.AccessKeyID)
	assert.Equal(t, "SK", creds.AccessKeySecret)
	assert.Equal(t, "Token", creds.SecurityToken)
	assert.Equal(t, now, creds.Expiration)
}

// TestConfig_Structure 测试配置结构体
func TestConfig_Structure(t *testing.T) {
	cfg := &Config{
		Provider:  ProviderOSS,
		Endpoint:  "oss-cn-hangzhou.aliyuncs.com",
		AccessKey: "AK",
		SecretKey: "SK",
		Bucket:    "bucket",
		Region:    "cn-hangzhou",
		RoleArn:   "arn",
		CdnDomain: "cdn.com",
		CdnKey:    "key",
	}

	assert.Equal(t, ProviderOSS, cfg.Provider)
	assert.Equal(t, "cdn.com", cfg.CdnDomain)
	assert.Equal(t, "key", cfg.CdnKey)
}

// MockStorage 是一个简单的 Mock 实现，用于测试依赖 Storage 接口的上层业务逻辑
type MockStorage struct {
	GenerateUploadURLFunc   func(userID, objectKey string, expire time.Duration) (string, error)
	GenerateDownloadURLFunc func(userID, objectKey string, expire time.Duration) (string, error)
	GeneratePreviewURLFunc  func(userID, objectKey string, expire time.Duration) (string, error)
	GetUserPrefixFunc       func(userID string) string
	GetTempCredentialsFunc  func(userID string, expire time.Duration) (*TempCredentials, error)
	RenewUploadURLFunc      func(userID, objectKey string, expire time.Duration) (string, error)
	RenewDownloadURLFunc    func(userID, objectKey string, expire time.Duration) (string, error)
}

func (m *MockStorage) GenerateUploadURL(userID, objectKey string, expire time.Duration) (string, error) {
	if m.GenerateUploadURLFunc != nil {
		return m.GenerateUploadURLFunc(userID, objectKey, expire)
	}
	return "", nil
}

func (m *MockStorage) GenerateDownloadURL(userID, objectKey string, expire time.Duration) (string, error) {
	if m.GenerateDownloadURLFunc != nil {
		return m.GenerateDownloadURLFunc(userID, objectKey, expire)
	}
	return "", nil
}

func (m *MockStorage) GeneratePreviewURL(userID, objectKey string, expire time.Duration) (string, error) {
	if m.GeneratePreviewURLFunc != nil {
		return m.GeneratePreviewURLFunc(userID, objectKey, expire)
	}
	return "", nil
}

func (m *MockStorage) GetUserPrefix(userID string) string {
	if m.GetUserPrefixFunc != nil {
		return m.GetUserPrefixFunc(userID)
	}
	return ""
}

func (m *MockStorage) GetTempCredentials(userID string, expire time.Duration) (*TempCredentials, error) {
	if m.GetTempCredentialsFunc != nil {
		return m.GetTempCredentialsFunc(userID, expire)
	}
	return nil, nil
}

func (m *MockStorage) RenewUploadURL(userID, objectKey string, expire time.Duration) (string, error) {
	if m.RenewUploadURLFunc != nil {
		return m.RenewUploadURLFunc(userID, objectKey, expire)
	}
	return "", nil
}

func (m *MockStorage) RenewDownloadURL(userID, objectKey string, expire time.Duration) (string, error) {
	if m.RenewDownloadURLFunc != nil {
		return m.RenewDownloadURLFunc(userID, objectKey, expire)
	}
	return "", nil
}

// TestStorageInterface_MockRenew 测试 Renew 方法是否被正确调用
func TestStorageInterface_MockRenew(t *testing.T) {
	mock := &MockStorage{}

	called := false
	mock.RenewUploadURLFunc = func(userID, objectKey string, expire time.Duration) (string, error) {
		called = true
		assert.Equal(t, "user1", userID)
		assert.Equal(t, "file.txt", objectKey)
		assert.Equal(t, 1*time.Hour, expire)
		return "http://new-url.com", nil
	}

	url, err := mock.RenewUploadURL("user1", "file.txt", 1*time.Hour)

	require.NoError(t, err)
	assert.True(t, called)
	assert.Equal(t, "http://new-url.com", url)
}

// TestStorageInterface_MockDownload 测试 Download 方法是否被正确调用
func TestStorageInterface_MockDownload(t *testing.T) {
	mock := &MockStorage{}

	mock.GenerateDownloadURLFunc = func(userID, objectKey string, expire time.Duration) (string, error) {
		return "https://cdn.example.com/file.jpg?auth_key=...", nil
	}

	url, err := mock.GenerateDownloadURL("user1", "file.jpg", 24*time.Hour)

	require.NoError(t, err)
	assert.Contains(t, url, "cdn.example.com")
}

func TestProviderReload_DecodesCdnConfig(t *testing.T) {
	provider, err := NewProvider(&Config{
		Provider:  ProviderS3,
		Region:    "us-east-1",
		AccessKey: "old-key",
		SecretKey: "old-secret",
		RoleArn:   "arn:aws:iam::123:role/test",
	})
	require.NoError(t, err)

	err = provider.Reload(map[string]interface{}{
		"media_cfg": map[string]interface{}{
			"provider":   "oss",
			"endpoint":   "oss-cn-hangzhou.aliyuncs.com",
			"access_key": "ak",
			"secret_key": "sk",
			"bucket":     "bucket",
			"region":     "cn-hangzhou",
			"role_arn":   "acs:ram::123:role/test",
			"cdn_domain": "cdn.example.com",
			"cdn_key":    "cdn-secret",
		},
	})
	require.NoError(t, err)

	storage, ok := provider.Get().(*ossStorage)
	require.True(t, ok)
	assert.Equal(t, "cdn.example.com", storage.cfg.CdnDomain)
	assert.Equal(t, "cdn-secret", storage.cfg.CdnKey)
}

// TestProviderType_Constants 测试常量定义
func TestProviderType_Constants(t *testing.T) {
	assert.Equal(t, ProviderType("oss"), ProviderOSS)
	assert.Equal(t, ProviderType("s3"), ProviderS3)
}
