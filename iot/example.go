package iot

import (
	"context"
	"fmt"
	"log"
	"time"
	//"comm/iot" // 假设您的 iot 模块路径是 comm/iot
)

// TODO：绑定和解绑时需要对用户名下所有设备进行权限更新，然后app端需要重新连接MQTT
func main() {
	ctx := context.Background()

	// --- 1. 配置 IoT 服务 ---
	// 请替换为您的实际 AWS 配置
	cfg := &Config{
		Region:    "ap-southeast-1", // 替换为您的 AWS Region
		AccountID: "123456789012",   // 替换为您的 AWS Account ID
		// 这是您在 AWS IAM 中创建的，用于 STS AssumeRole 的基础角色 ARN。
		// 您的后端服务需要有权限 Assume 这个角色。
		ScopedAccessRoleARN: "arn:aws:iam::123456789012:role/iot-scoped-access-role", // 替换为您的角色 ARN
		// 临时凭证的有效期，例如 900 秒 (15 分钟)
		CredentialDurationSeconds: 900,
	}

	// --- 2. 初始化 IoT 服务 ---
	iotService, err := NewService(ctx, cfg)
	if err != nil {
		log.Fatalf("Failed to initialize IoT service: %v", err)
	}
	fmt.Printf("IoT Service initialized. Endpoint: %s\n", iotService.Endpoint())

	// --- 3. 定义用户 ID 和允许访问的 Topic 列表 ---
	// 假设这是您从用户认证系统获取的用户唯一标识符
	userID := "user-alice-123"

	allowedTopics := TopicPermission{
		Publish: []string{
			fmt.Sprintf("devices/%s/commands", userID),
		},

		Subscribe: []string{
			fmt.Sprintf("devices/%s/data", userID),
		},
	}

	// --- 4. 获取临时凭证 ---
	fmt.Printf("\nRequesting temporary credentials for user '%s' with topics: %v\n", userID, allowedTopics)
	creds, err := iotService.GetTemporaryScopedCredentials(ctx, userID, allowedTopics)
	if err != nil {
		log.Fatalf("Failed to get temporary scoped credentials: %v", err)
	}

	// --- 5. 打印生成的凭证 ---
	fmt.Println("\n--- Generated Temporary Credentials ---")
	fmt.Printf("Access Key ID:     %s\n", creds.Credentials.AccessKeyID)
	fmt.Printf("Secret Access Key: %s\n", creds.Credentials.SecretAccessKey)
	fmt.Printf("Session Token:     %s\n", creds.Credentials.SessionToken)
	fmt.Printf("Expiration:        %s (in %s)\n", creds.Credentials.Expiration.Format(time.RFC3339), time.Until(creds.Credentials.Expiration).Round(time.Second))
	fmt.Println("------------------------------------")

	fmt.Println("\nThese credentials can now be sent to the client application.")
	fmt.Println("The client application should use these credentials to connect to AWS IoT Core via WebSocket.")
	fmt.Println("Remember to implement a credential refresh mechanism in the client application.")

	// --- 6. 演示 Publish 功能 (可选) ---
	// 假设后端需要向某个设备发布消息
	publishTopic := fmt.Sprintf("devices/%s/commands", userID)
	payload := []byte(fmt.Sprintf(`{"command": "turn_on", "timestamp": %d}`, time.Now().Unix()))

	fmt.Printf("\nAttempting to publish message to topic '%s'...\n", publishTopic)
	err = iotService.Publish(ctx, publishTopic, payload)
	if err != nil {
		log.Printf("Failed to publish message: %v", err)
	} else {
		fmt.Printf("Successfully published message to '%s': %s\n", publishTopic, string(payload))
	}
}
