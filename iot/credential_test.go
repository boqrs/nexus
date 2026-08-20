package iot

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	ststypes "github.com/aws/aws-sdk-go-v2/service/sts/types"
)

type fakeSTS struct {
	output *sts.AssumeRoleOutput
	err    error

	input *sts.AssumeRoleInput
}

func (f *fakeSTS) AssumeRole(
	ctx context.Context,
	in *sts.AssumeRoleInput,
	optFns ...func(*sts.Options),
) (*sts.AssumeRoleOutput, error) {
	f.input = in

	if f.err != nil {
		return nil, f.err
	}

	return f.output, nil
}

func TestGetTemporaryScopedCredentials(t *testing.T) {
	expiration := time.Now().Add(time.Hour)

	fake := &fakeSTS{
		output: &sts.AssumeRoleOutput{
			Credentials: &ststypes.Credentials{
				AccessKeyId:     aws.String("ak"),
				SecretAccessKey: aws.String("sk"),
				SessionToken:    aws.String("token"),
				Expiration:      aws.Time(expiration),
			},
		},
	}

	svc := &Service{
		cfg: &Config{
			AccountID:                 "123456789012",
			Region:                    "ap-southeast-1",
			Endpoint:                  "xxxxx.iot.amazonaws.com",
			ScopedAccessRoleARN:       "arn:aws:iam::123456789012:role/test",
			CredentialDurationSeconds: 3600,
		},
		stsClient: fake,
	}

	info, err := svc.GetTemporaryScopedCredentials(
		context.Background(),
		"client-001",
		TopicPermission{
			Publish: []string{
				"printer/001/command",
			},
			Subscribe: []string{
				"printer/001/status",
			},
		},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if info.ClientID != "client-001" {
		t.Fatalf("unexpected clientID")
	}

	if info.Endpoint != "xxxxx.iot.amazonaws.com" {
		t.Fatalf("unexpected endpoint")
	}

	if info.Region != "ap-southeast-1" {
		t.Fatalf("unexpected region")
	}

	if info.Credentials.AccessKeyID != "ak" {
		t.Fatalf("unexpected access key")
	}

	if info.Credentials.SecretAccessKey != "sk" {
		t.Fatalf("unexpected secret key")
	}

	if info.Credentials.SessionToken != "token" {
		t.Fatalf("unexpected token")
	}

	if !info.Credentials.Expiration.Equal(expiration) {
		t.Fatalf("unexpected expiration")
	}

	if fake.input == nil {
		t.Fatalf("AssumeRole not called")
	}

	if aws.ToString(fake.input.RoleArn) != svc.cfg.ScopedAccessRoleARN {
		t.Fatalf("unexpected role arn")
	}

	if aws.ToInt32(fake.input.DurationSeconds) != 3600 {
		t.Fatalf("unexpected duration")
	}

	if fake.input.Policy == nil {
		t.Fatalf("policy should not be nil")
	}
}

func TestGetTemporaryScopedCredentials_AssumeRoleError(t *testing.T) {
	fake := &fakeSTS{
		err: errors.New("sts error"),
	}

	svc := &Service{
		cfg: &Config{
			AccountID:                 "123456789012",
			Region:                    "ap-southeast-1",
			Endpoint:                  "xxxxx.iot.amazonaws.com",
			ScopedAccessRoleARN:       "arn:aws:iam::123456789012:role/test",
			CredentialDurationSeconds: 3600,
		},
		stsClient: fake,
	}

	_, err := svc.GetTemporaryScopedCredentials(
		context.Background(),
		"client-001",
		TopicPermission{},
	)

	if err == nil {
		t.Fatal("expected error")
	}
}

func TestGetTemporaryScopedCredentials_EmptyClientID(t *testing.T) {
	svc := &Service{}

	_, err := svc.GetTemporaryScopedCredentials(
		context.Background(),
		"",
		TopicPermission{},
	)

	if err == nil {
		t.Fatal("expected error")
	}
}
