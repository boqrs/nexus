package iot

import "time"

// Credentials represents temporary AWS credentials returned by STS.
//
// These credentials are intended for App clients to establish
// MQTT over WebSocket connections to AWS IoT Core.
type Credentials struct {
	// AWS access key ID.
	AccessKeyID string `json:"accessKeyId"`

	// AWS secret access key.
	SecretAccessKey string `json:"secretAccessKey"`

	// AWS session token.
	SessionToken string `json:"sessionToken"`

	// Credential expiration time.
	Expiration time.Time `json:"expiration"`
}

// ConnectInfo contains everything an App needs to establish
// a MQTT over WebSocket connection to AWS IoT Core.
type ConnectInfo struct {
	// AWS IoT endpoint.
	//
	// Example:
	// xxxxx-ats.iot.ap-southeast-1.amazonaws.com
	Endpoint string `json:"endpoint"`

	// AWS region.
	Region string `json:"region"`

	// MQTT ClientID.
	//
	// This value MUST be used when creating the MQTT connection.
	ClientID string `json:"clientId"`

	// Temporary AWS credentials.
	Credentials Credentials `json:"credentials"`
}

// TopicPermission defines MQTT topic permissions granted
// to a temporary App session.
//
// Publish and Subscribe are intentionally separated because
// they map to different IAM actions.
type TopicPermission struct {
	// Topics that the client is allowed to publish.
	//
	// Example:
	// printer/123/command
	Publish []string `json:"publish"`

	// Topic filters that the client is allowed to subscribe.
	//
	// Example:
	// printer/123/status
	// printer/123/event
	Subscribe []string `json:"subscribe"`
}
