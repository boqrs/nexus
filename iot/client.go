package iot

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/iot"
	iotdataplane "github.com/aws/aws-sdk-go-v2/service/iotdataplane"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

// STSAPI defines the minimal STS API used by this package.
type STSAPI interface {
	AssumeRole(
		ctx context.Context,
		params *sts.AssumeRoleInput,
		optFns ...func(*sts.Options),
	) (*sts.AssumeRoleOutput, error)
}

// IoTControlAPI defines the minimal AWS IoT Control Plane API used by this package.
type IoTControlAPI interface {
	DescribeEndpoint(
		ctx context.Context,
		params *iot.DescribeEndpointInput,
		optFns ...func(*iot.Options),
	) (*iot.DescribeEndpointOutput, error)
}

// IoTDataPlaneAPI defines the minimal AWS IoT Data Plane API used by this package.
type IoTDataPlaneAPI interface {
	Publish(
		ctx context.Context,
		params *iotdataplane.PublishInput,
		optFns ...func(*iotdataplane.Options),
	) (*iotdataplane.PublishOutput, error)
}
