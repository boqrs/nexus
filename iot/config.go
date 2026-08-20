package iot

import "fmt"

// Config contains all configuration required by the AWS IoT service.
type Config struct {
	// AWS region.
	Region string `json:"region" yaml:"region"`

	// AWS account ID.
	//
	// Example:
	// 123456789012
	AccountID string `json:"account_id" yaml:"account_id"`

	// AWS IoT Data Endpoint.
	//
	// Example:
	// a1b2c3d4e5-ats.iot.ap-southeast-1.amazonaws.com
	//
	// If empty, the SDK will automatically resolve the endpoint
	// using the AWS IoT DescribeEndpoint API during initialization.
	Endpoint string `json:"endpoint" yaml:"endpoint"`

	// IAM Role ARN used by STS AssumeRole.
	//
	// The temporary credentials returned to App clients will be scoped
	// by an inline session policy generated at runtime.
	ScopedAccessRoleARN string `json:"scoped_access_role_arn" yaml:"scoped_access_role_arn"`

	// Duration of temporary credentials.
	//
	// Default:
	// 3600 (1 hour)
	CredentialDurationSeconds int32 `json:"credential_duration_seconds" yaml:"credential_duration_seconds"`
}

// SetDefault fills optional configuration values.
func (c *Config) SetDefault() {
	if c.Region == "" {
		c.Region = "ap-southeast-1"
	}

	if c.CredentialDurationSeconds == 0 {
		c.CredentialDurationSeconds = 3600
	}
}

// Validate validates the configuration.
func (c *Config) Validate() error {
	if c.Region == "" {
		return fmt.Errorf("region is required")
	}

	if c.AccountID == "" {
		return fmt.Errorf("account_id is required")
	}

	if c.ScopedAccessRoleARN == "" {
		return fmt.Errorf("scoped_access_role_arn is required")
	}

	return nil
}
