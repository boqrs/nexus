package iot

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"

	iot "github.com/aws/aws-sdk-go-v2/service/iot"
	iotdataplane "github.com/aws/aws-sdk-go-v2/service/iotdataplane"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

// Service provides AWS IoT Core operations.
type Service struct {
	cfg *Config

	// Runtime IoT endpoint.
	endpoint string

	// AWS clients.
	iotClient IoTControlAPI

	iotDataPlane IoTDataPlaneAPI

	stsClient STSAPI
}

// NewService creates a production AWS IoT service.
func NewService(
	ctx context.Context,
	cfg *Config,
) (*Service, error) {
	cfg.SetDefault()

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(
		ctx,
		awsconfig.WithRegion(
			cfg.Region,
		),
	)
	if err != nil {
		return nil, fmt.Errorf(
			"load aws config: %w",
			err,
		)
	}

	iotClient := iot.NewFromConfig(
		awsCfg,
	)

	endpoint := cfg.Endpoint

	if endpoint == "" {

		endpoint, err = discoverEndpoint(
			ctx,
			iotClient,
		)
		if err != nil {
			return nil, err
		}

	}

	dataPlane := iotdataplane.NewFromConfig(
		awsCfg,
		func(o *iotdataplane.Options) {
			o.BaseEndpoint = aws.String(
				"https://" + endpoint,
			)
		},
	)

	return &Service{
		cfg: cfg,

		endpoint: endpoint,

		iotClient: iotClient,

		iotDataPlane: dataPlane,

		stsClient: sts.NewFromConfig(
			awsCfg,
		),
	}, nil
}

// NewServiceWithClients creates service with injected clients.
//
// Used for:
//   - unit tests
//   - LocalStack
//   - custom implementations
func NewServiceWithClients(
	cfg *Config,
	iotClient IoTControlAPI,
	dataPlane IoTDataPlaneAPI,
	stsClient STSAPI,
	endpoint string,
) *Service {
	cfg.SetDefault()

	return &Service{
		cfg: cfg,

		endpoint: endpoint,

		iotClient: iotClient,

		iotDataPlane: dataPlane,

		stsClient: stsClient,
	}
}

// Endpoint returns IoT MQTT endpoint.
func (s *Service) Endpoint() string {
	return s.endpoint
}

// discoverEndpoint gets ATS endpoint.
func discoverEndpoint(
	ctx context.Context,
	client IoTControlAPI,
) (string, error) {
	output, err := client.DescribeEndpoint(
		ctx,
		&iot.DescribeEndpointInput{
			EndpointType: aws.String(
				"iot:Data-ATS",
			),
		},
	)
	if err != nil {
		return "",
			fmt.Errorf(
				"describe endpoint: %w",
				err,
			)
	}

	if output.EndpointAddress == nil {
		return "",
			fmt.Errorf(
				"iot endpoint empty",
			)
	}

	return aws.ToString(
		output.EndpointAddress,
	), nil
}
