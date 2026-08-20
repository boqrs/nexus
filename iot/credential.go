package iot

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

// GetTemporaryScopedCredentials creates temporary AWS credentials
// that are scoped to a specific MQTT client and topic permissions.
//
// The returned ConnectInfo contains everything required by an App
// to establish a MQTT over WebSocket connection.
//
// The permission granted is the intersection of:
//
//   - ScopedAccessRoleARN
//   - Generated inline session policy
func (s *Service) GetTemporaryScopedCredentials(
	ctx context.Context,
	clientID string,
	permission TopicPermission,
) (*ConnectInfo, error) {
	if clientID == "" {
		return nil, fmt.Errorf("clientID is empty")
	}

	policy, err := s.generateScopedPolicy(
		clientID,
		permission,
	)
	if err != nil {
		return nil, err
	}

	result, err := s.stsClient.AssumeRole(ctx, &sts.AssumeRoleInput{
		RoleArn: aws.String(s.cfg.ScopedAccessRoleARN),

		RoleSessionName: aws.String(clientID),

		DurationSeconds: aws.Int32(
			s.cfg.CredentialDurationSeconds,
		),

		Policy: aws.String(policy),
	})
	if err != nil {
		return nil, fmt.Errorf("assume role: %w", err)
	}

	return &ConnectInfo{
		Endpoint: s.cfg.Endpoint,
		Region:   s.cfg.Region,
		ClientID: clientID,

		Credentials: Credentials{
			AccessKeyID:     aws.ToString(result.Credentials.AccessKeyId),
			SecretAccessKey: aws.ToString(result.Credentials.SecretAccessKey),
			SessionToken:    aws.ToString(result.Credentials.SessionToken),
			Expiration:      aws.ToTime(result.Credentials.Expiration),
		},
	}, nil
}
