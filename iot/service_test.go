package iot

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iot"
)

type fakeIoTControl struct {
	called bool

	output *iot.DescribeEndpointOutput

	err error
}

func (f *fakeIoTControl) DescribeEndpoint(
	ctx context.Context,
	input *iot.DescribeEndpointInput,
	optFns ...func(*iot.Options),
) (
	*iot.DescribeEndpointOutput,
	error,
) {
	f.called = true

	if f.err != nil {
		return nil, f.err
	}

	return f.output, nil
}

func TestNewServiceWithClients(t *testing.T) {
	cfg := &Config{
		Region: "ap-southeast-1",

		AccountID: "123456789012",

		ScopedAccessRoleARN: "arn:aws:iam::123456789012:role/test",
	}

	fakeIoT := &fakeIoTControl{}

	svc := NewServiceWithClients(

		cfg,

		fakeIoT,

		nil,

		nil,

		"xxxxx.iot.amazonaws.com",
	)

	if svc == nil {
		t.Fatal(
			"service should not be nil",
		)
	}

	if svc.Endpoint() != "xxxxx.iot.amazonaws.com" {
		t.Fatalf(
			"endpoint=%s",
			svc.Endpoint(),
		)
	}

	if svc.iotClient != fakeIoT {
		t.Fatal(
			"iot client injection failed",
		)
	}
}

func TestDiscoverEndpoint(t *testing.T) {
	fake := &fakeIoTControl{
		output: &iot.DescribeEndpointOutput{
			EndpointAddress: aws.String(
				"abc.iot.amazonaws.com",
			),
		},
	}

	endpoint, err := discoverEndpoint(
		context.Background(),
		fake,
	)
	if err != nil {
		t.Fatal(err)
	}

	if endpoint != "abc.iot.amazonaws.com" {
		t.Fatalf(
			"endpoint=%s",
			endpoint,
		)
	}

	if !fake.called {
		t.Fatal(
			"DescribeEndpoint not called",
		)
	}
}

func TestServiceEndpoint(t *testing.T) {
	svc := &Service{
		endpoint: "test.iot.amazonaws.com",
	}

	if svc.Endpoint() != "test.iot.amazonaws.com" {
		t.Fatal(
			"unexpected endpoint",
		)
	}
}
