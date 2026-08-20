package media

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url" // 需要添加这个
	"strings"
	"time"

	// 阿里云 SDK
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk"
	// 【关键】使用别名 aliyunCreds 避免与 aws credentials 包名冲突
	aliyunCreds "github.com/aliyun/alibaba-cloud-sdk-go/sdk/auth/credentials"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	aliyunsts "github.com/aliyun/alibaba-cloud-sdk-go/services/sts"

	// 阿里云 OSS SDK
	"github.com/aliyun/aliyun-oss-go-sdk/oss"

	// AWS SDK
	"github.com/aws/aws-sdk-go/aws"
	// AWS credentials 保持默认名称
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	awssts "github.com/aws/aws-sdk-go/service/sts"
)

// ProviderType 表示后端存储类型
type ProviderType string

const (
	ProviderOSS ProviderType = "oss"
	ProviderS3  ProviderType = "s3"
)

// Config 用于初始化存储客户端
type Config struct {
	Provider  ProviderType `json:"provider" yaml:"provider" mapstructure:"provider"`
	Endpoint  string       `json:"endpoint" yaml:"endpoint" mapstructure:"endpoint"`
	AccessKey string       `json:"access_key" yaml:"access_key" mapstructure:"access_key"`
	SecretKey string       `json:"secret_key" yaml:"secret_key" mapstructure:"secret_key"`
	Bucket    string       `json:"bucket" yaml:"bucket" mapstructure:"bucket"`
	Region    string       `json:"region" yaml:"region" mapstructure:"region"` // only for s3
	RoleArn   string       `json:"role_arn" yaml:"role_arn"`                   // STS角色ARN (可选，用于更安全的授权)

	// --- CDN 配置 (新增) ---
	// CdnDomain: CDN 加速域名，例如 cdn.example.com。如果为空，下载将回退到源站签名URL(受限于STS有效期)
	CdnDomain string `json:"cdn_domain" yaml:"cdn_domain" mapstructure:"cdn_domain"`
	// CdnKey: CDN 鉴权私钥 (在阿里云/腾讯云/AWS CloudFront 控制台配置)。仅当使用 CDN 鉴权时需要。
	CdnKey string `json:"cdn_key" yaml:"cdn_key" mapstructure:"cdn_key"`
}

// TempCredentials 临时凭证
type TempCredentials struct {
	AccessKeyID     string
	AccessKeySecret string
	SecurityToken   string
	Expiration      time.Time
}

// Storage 抽象接口，屏蔽OSS/S3差异，支持多租户
type Storage interface {
	// GenerateUploadURL 创建指定object的临时上传URL（多租户隔离，走STS）
	GenerateUploadURL(userID, objectKey string, expire time.Duration) (string, error)
	// GenerateDownloadURL 创建指定object的临时下载URL（优先走CDN，支持长有效期）
	GenerateDownloadURL(userID, objectKey string, expire time.Duration) (string, error)
	// GeneratePreviewURL 创建预览URL（用于图片、文档等，优先走CDN）
	GeneratePreviewURL(userID, objectKey string, expire time.Duration) (string, error)
	// GetUserPrefix 获取用户目录前缀
	GetUserPrefix(userID string) string
	// GetTempCredentials 获取用户的临时凭证
	GetTempCredentials(userID string, expire time.Duration) (*TempCredentials, error)
	// --- 续期方法 ---
	// RenewUploadURL 为上传链接续期，返回新的URL
	RenewUploadURL(userID, objectKey string, expire time.Duration) (string, error)
	// RenewDownloadURL 为下载链接续期，返回新的URL
	RenewDownloadURL(userID, objectKey string, expire time.Duration) (string, error)
}

// NewStorage 根据配置构建Storage实例
func NewStorage(cfg *Config) (Storage, error) {
	switch cfg.Provider {
	case ProviderOSS:
		return newOSSStorage(cfg)
	case ProviderS3:
		return newS3Storage(cfg)
	default:
		return nil, fmt.Errorf("unsupported provider: %s", cfg.Provider)
	}
}

// buildObjectKey 构建带用户前缀的对象键
func buildObjectKey(userID, objectKey string) string {
	objectKey = strings.TrimPrefix(objectKey, "/")
	return userID + "/" + objectKey
}

// --- 通用工具函数 ---

// generateAliyunCdnSignedURL 生成阿里云 CDN Type A 签名 URL
// 格式: http://Domain/Filename?auth_key=timestamp-rand-uid-md5hash
func generateAliyunCdnSignedURL(domain, objectKey, key string, expire time.Duration) string {
	timestamp := time.Now().Add(expire).Unix()
	randStr := "0" // 随机串，简单场景可固定为0
	uid := "0"     // 用户ID，简单场景可固定为0

	// 构造签名字符串: /filename-timestamp-rand-uid-key
	// 注意：objectKey 应该已经是相对路径，不包含域名
	signStr := fmt.Sprintf("/%s-%d-%s-%s-%s", objectKey, timestamp, randStr, uid, key)

	// 计算 MD5: md5(signStr)
	hash := md5.Sum([]byte(signStr))
	md5Hash := fmt.Sprintf("%x", hash)

	// 最终 URL
	return fmt.Sprintf("https://%s/%s?auth_key=%d-%s-%s-%s",
		domain,
		objectKey,
		timestamp,
		randStr,
		uid,
		md5Hash,
	)
}

// --- 阿里云OSS实现 ---

type ossStorage struct {
	cfg       *Config
	stsClient *aliyunsts.Client
}

// newOSSStorage 初始化阿里云 OSS 存储客户端，强制使用 HTTPS
func newOSSStorage(cfg *Config) (Storage, error) {
	// 创建自定义 Transport
	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout:   15 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
	}

	// 基础配置
	sdkConfig := sdk.NewConfig().
		WithScheme("HTTPS").
		WithTimeout(60 * time.Second)

	cred := aliyunCreds.NewAccessKeyCredential(cfg.AccessKey, cfg.SecretKey)

	stsClient, err := aliyunsts.NewClientWithOptions(cfg.Region, sdkConfig, cred)
	if err != nil {
		return nil, fmt.Errorf("failed to create STS client: %w", err)
	}

	// 关键：设置自定义 Transport
	stsClient.SetTransport(transport)

	// 额外设置超时
	stsClient.SetConnectTimeout(30 * time.Second)
	stsClient.SetReadTimeout(60 * time.Second)

	return &ossStorage{
		cfg:       cfg,
		stsClient: stsClient,
	}, nil
}

func (o *ossStorage) GetUserPrefix(userID string) string {
	return userID + "/"
}

func (o *ossStorage) GetTempCredentials(userID string, expire time.Duration) (*TempCredentials, error) {
	if userID == "" {
		return nil, fmt.Errorf("user_id is required")
	}

	// 【修复】钳制有效期范围：阿里云 STS 要求 15min - 1hr
	const minExpire = 15 * time.Minute
	const maxExpire = 1 * time.Hour

	durationSeconds := int(expire.Seconds())
	if durationSeconds < int(minExpire.Seconds()) {
		durationSeconds = int(minExpire.Seconds())
	} else if durationSeconds > int(maxExpire.Seconds()) {
		durationSeconds = int(maxExpire.Seconds())
	}

	// 构建精细的权限策略，只允许访问用户的目录
	policy := map[string]interface{}{
		"Version": "2012-10-17", // 必须是这个
		"Statement": []map[string]interface{}{
			{
				"Effect": "Allow",
				"Action": []string{
					"oss:GetObject",
					"oss:PutObject",
					"oss:DeleteObject",
				},
				"Resource": []string{
					fmt.Sprintf("acs:oss:*:*:%s/%s/*", o.cfg.Bucket, userID),
					fmt.Sprintf("acs:oss:*:*:%s", o.cfg.Bucket),
				},
			},
		},
	}

	policyBytes, err := json.Marshal(policy)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal policy: %w", err)
	}

	request := aliyunsts.CreateAssumeRoleRequest()
	request.RoleArn = o.cfg.RoleArn
	request.RoleSessionName = fmt.Sprintf("user-%s-session", userID)
	// 使用钳制后的时间
	request.DurationSeconds = requests.NewInteger(durationSeconds)
	request.Policy = string(policyBytes)

	response, err := o.stsClient.AssumeRole(request)
	if err != nil {
		return nil, fmt.Errorf("failed to assume role: %w", err)
	}

	expiration, err := time.Parse(time.RFC3339, response.Credentials.Expiration)
	if err != nil {
		return nil, fmt.Errorf("failed to parse expiration: %w", err)
	}

	return &TempCredentials{
		AccessKeyID:     response.Credentials.AccessKeyId,
		AccessKeySecret: response.Credentials.AccessKeySecret,
		SecurityToken:   response.Credentials.SecurityToken,
		Expiration:      expiration,
	}, nil
}

func (o *ossStorage) GenerateUploadURL(userID, objectKey string, expire time.Duration) (string, error) {
	// 获取临时凭证
	tempCred, err := o.GetTempCredentials(userID, expire)
	if err != nil {
		return "", fmt.Errorf("failed to get temp credentials: %w", err)
	}

	// 使用临时凭证创建OSS客户端
	client, err := oss.New(o.cfg.Endpoint, tempCred.AccessKeyID, tempCred.AccessKeySecret, oss.SecurityToken(tempCred.SecurityToken))
	if err != nil {
		return "", fmt.Errorf("failed to create OSS client with temp credentials: %w", err)
	}

	bucket, err := client.Bucket(o.cfg.Bucket)
	if err != nil {
		return "", fmt.Errorf("failed to get OSS bucket: %w", err)
	}

	fullKey := buildObjectKey(userID, objectKey)

	// 生成PUT预签名URL

	signedURL, err := bucket.SignURL(fullKey, oss.HTTPPut, int64(expire.Seconds()))
	if err != nil {
		return "", fmt.Errorf("failed to generate OSS upload URL: %w", err)
	}

	// 【关键修复】阿里云 OSS 的 STS 预签名 URL 必须包含 security-token 参数
	// 因为 SignURL 方法不会自动添加它
	parsedURL, err := url.Parse(signedURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse signed URL: %w", err)
	}

	query := parsedURL.Query()
	// 只有使用 STS 临时凭证时才需要添加
	if tempCred.SecurityToken != "" {
		query.Set("security-token", tempCred.SecurityToken)
	}
	parsedURL.RawQuery = query.Encode()

	return parsedURL.String(), nil
}

func (o *ossStorage) GenerateDownloadURL(userID, objectKey string, expire time.Duration) (string, error) {
	fullKey := buildObjectKey(userID, objectKey)

	// 优先使用 CDN
	if o.cfg.CdnDomain != "" && o.cfg.CdnKey != "" {
		url := generateAliyunCdnSignedURL(o.cfg.CdnDomain, fullKey, o.cfg.CdnKey, expire)
		return url, nil
	}

	// 降级方案：OSS 原生签名 URL
	tempCred, err := o.GetTempCredentials(userID, expire)
	if err != nil {
		return "", fmt.Errorf("failed to get temp credentials: %w", err)
	}

	client, err := oss.New(o.cfg.Endpoint, tempCred.AccessKeyID, tempCred.AccessKeySecret, oss.SecurityToken(tempCred.SecurityToken))
	if err != nil {
		return "", fmt.Errorf("failed to create OSS client with temp credentials: %w", err)
	}

	bucket, err := client.Bucket(o.cfg.Bucket)
	if err != nil {
		return "", fmt.Errorf("failed to get OSS bucket: %w", err)
	}

	signedURL, err := bucket.SignURL(fullKey, oss.HTTPGet, int64(expire.Seconds()))
	if err != nil {
		return "", fmt.Errorf("failed to generate OSS download URL: %w", err)
	}

	// 【关键修复】添加 security-token 参数
	parsedURL, err := url.Parse(signedURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse signed URL: %w", err)
	}

	parsedURL.Path, err = url.PathUnescape(parsedURL.Path)
	if err != nil {
		return "", err
	}

	query := parsedURL.Query()
	if tempCred.SecurityToken != "" {
		query.Set("security-token", tempCred.SecurityToken)
	}
	parsedURL.RawQuery = query.Encode()

	return parsedURL.String(), nil
}

func (o *ossStorage) GeneratePreviewURL(userID, objectKey string, expire time.Duration) (string, error) {
	// 预览通常也走 CDN
	return o.GenerateDownloadURL(userID, objectKey, expire)
}

func (o *ossStorage) RenewUploadURL(userID, objectKey string, expire time.Duration) (string, error) {
	return o.GenerateUploadURL(userID, objectKey, expire)
}

func (o *ossStorage) RenewDownloadURL(userID, objectKey string, expire time.Duration) (string, error) {
	// CDN URL 重新生成成本极低，直接重新生成即可
	return o.GenerateDownloadURL(userID, objectKey, expire)
}

// --- AWS S3实现 ---

type s3Storage struct {
	cfg       *Config
	stsClient *awssts.STS
}

func newS3Storage(cfg *Config) (Storage, error) {
	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String(cfg.Region),
		Credentials: credentials.NewStaticCredentials(cfg.AccessKey, cfg.SecretKey, ""),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create AWS session: %w", err)
	}

	stsClient := awssts.New(sess)

	return &s3Storage{
		cfg:       cfg,
		stsClient: stsClient,
	}, nil
}

func (s *s3Storage) GetUserPrefix(userID string) string {
	return userID + "/"
}

func (s *s3Storage) GetTempCredentials(userID string, expire time.Duration) (*TempCredentials, error) {
	if userID == "" {
		return nil, fmt.Errorf("user_id is required")
	}

	if s.cfg.RoleArn == "" {
		return nil, fmt.Errorf("RoleArn is required for STS temporary credentials")
	}

	// 【修复】钳制有效期范围：AWS STS 默认也是 15min - 1hr (除非角色配置了更长的 MaxSessionDuration)
	const minExpire = 15 * time.Minute
	const maxExpire = 1 * time.Hour

	durationSeconds := int64(expire.Seconds())
	if durationSeconds < int64(minExpire.Seconds()) {
		durationSeconds = int64(minExpire.Seconds())
	} else if durationSeconds > int64(maxExpire.Seconds()) {
		durationSeconds = int64(maxExpire.Seconds())
	}

	// 构建精细的权限策略，只允许访问用户的目录
	policy := map[string]interface{}{
		"Version": "2012-10-17",
		"Statement": []map[string]interface{}{
			{
				"Effect": "Allow",
				"Action": []string{
					"s3:GetObject",
					"s3:PutObject",
					"s3:DeleteObject",
				},
				"Resource": []string{
					fmt.Sprintf("arn:aws:s3:::%s/%s/*", s.cfg.Bucket, userID),
				},
			},
		},
	}

	policyBytes, err := json.Marshal(policy)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal policy: %w", err)
	}

	input := &awssts.AssumeRoleInput{
		RoleArn:         aws.String(s.cfg.RoleArn),
		RoleSessionName: aws.String(fmt.Sprintf("media-session-%s", userID)),
		DurationSeconds: aws.Int64(durationSeconds), // 使用钳制后的时间
		Policy:          aws.String(string(policyBytes)),
	}

	result, err := s.stsClient.AssumeRole(input)
	if err != nil {
		return nil, fmt.Errorf("failed to assume role: %w", err)
	}

	return &TempCredentials{
		AccessKeyID:     *result.Credentials.AccessKeyId,
		AccessKeySecret: *result.Credentials.SecretAccessKey,
		SecurityToken:   *result.Credentials.SessionToken,
		Expiration:      *result.Credentials.Expiration,
	}, nil
}

func (s *s3Storage) GenerateUploadURL(userID, objectKey string, expire time.Duration) (string, error) {
	// 获取临时凭证
	tempCred, err := s.GetTempCredentials(userID, expire)
	if err != nil {
		return "", fmt.Errorf("failed to get temp credentials: %w", err)
	}

	// 使用临时凭证创建S3客户端
	sess, err := session.NewSession(&aws.Config{
		Region:           aws.String(s.cfg.Region),
		Credentials:      credentials.NewStaticCredentials(tempCred.AccessKeyID, tempCred.AccessKeySecret, tempCred.SecurityToken),
		Endpoint:         aws.String(s.cfg.Endpoint),
		S3ForcePathStyle: aws.Bool(true), // 使用path-style URLs
	})
	if err != nil {
		return "", fmt.Errorf("failed to create session with temp credentials: %w", err)
	}

	client := s3.New(sess)

	fullKey := buildObjectKey(userID, objectKey)

	req, _ := client.PutObjectRequest(&s3.PutObjectInput{
		Bucket: aws.String(s.cfg.Bucket),
		Key:    aws.String(fullKey),
	})

	// 生成预签名URL
	signedURL, err := req.Presign(expire)
	if err != nil {
		return "", fmt.Errorf("failed to generate S3 upload URL: %w", err)
	}

	return signedURL, nil
}

func (s *s3Storage) GenerateDownloadURL(userID, objectKey string, expire time.Duration) (string, error) {
	fullKey := buildObjectKey(userID, objectKey)

	// 优先使用 CDN (假设 CDN 鉴权逻辑与阿里云类似，或者使用 CloudFront Signed URLs)
	// 注意：这里简化处理，如果配置了 CdnDomain 和 CdnKey，我们暂时沿用阿里云的签名逻辑作为示例。
	// 实际生产中，AWS CloudFront 的签名逻辑不同，需要引入 cloudfront/sign 包。
	// 如果未配置 CDN，则回退到 S3 预签名 URL。

	if s.cfg.CdnDomain != "" && s.cfg.CdnKey != "" {
		// 提示：如果是 AWS CloudFront，请使用 github.com/aws/aws-sdk-go/service/cloudfront/sign 包生成签名
		// 这里为了代码简洁，暂且复用阿里云逻辑演示“走CDN”的概念，实际需替换为 CloudFront 签名逻辑
		// 或者，如果 CDN 只是简单的域名映射且 Bucket 公开读（不推荐），则直接拼接域名。

		// 假设 CDN 只是域名替换，且 Bucket 权限允许 CDN 回源（通常需要签名）
		// 此处仅为占位，实际 AWS 场景建议集成 CloudFront Signer
		return "", fmt.Errorf("AWS CloudFront signed URL generation not implemented in this snippet. Please use cloudfront/sign package.")
	}

	// 降级方案：S3 原生预签名 URL
	tempCred, err := s.GetTempCredentials(userID, expire)
	if err != nil {
		return "", fmt.Errorf("failed to get temp credentials: %w", err)
	}

	sess, err := session.NewSession(&aws.Config{
		Region:           aws.String(s.cfg.Region),
		Credentials:      credentials.NewStaticCredentials(tempCred.AccessKeyID, tempCred.AccessKeySecret, tempCred.SecurityToken),
		Endpoint:         aws.String(s.cfg.Endpoint),
		S3ForcePathStyle: aws.Bool(true),
	})
	if err != nil {
		return "", fmt.Errorf("failed to create session with temp credentials: %w", err)
	}

	client := s3.New(sess)

	req, _ := client.GetObjectRequest(&s3.GetObjectInput{
		Bucket: aws.String(s.cfg.Bucket),
		Key:    aws.String(fullKey),
	})

	signedURL, err := req.Presign(expire)
	if err != nil {
		return "", fmt.Errorf("failed to generate S3 download URL: %w", err)
	}

	return signedURL, nil
}

func (s *s3Storage) GeneratePreviewURL(userID, objectKey string, expire time.Duration) (string, error) {
	return s.GenerateDownloadURL(userID, objectKey, expire)
}

func (s *s3Storage) RenewUploadURL(userID, objectKey string, expire time.Duration) (string, error) {
	return s.GenerateUploadURL(userID, objectKey, expire)
}

func (s *s3Storage) RenewDownloadURL(userID, objectKey string, expire time.Duration) (string, error) {
	return s.GenerateDownloadURL(userID, objectKey, expire)
}
