package iot

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	iotdataplane "github.com/aws/aws-sdk-go-v2/service/iotdataplane"
)

// Publish sends a raw payload to an MQTT topic.
//
// This method is intended for backend-to-device communication.
//
// Example:
//
//	err := svc.Publish(ctx,
//	    "printer/123/command",
//	    []byte(`{"cmd":"pause"}`),
//	)
func (s *Service) Publish(
	ctx context.Context,
	topic string,
	payload []byte,
) error {
	if topic == "" {
		return fmt.Errorf("topic is empty")
	}

	_, err := s.iotDataPlane.Publish(ctx, &iotdataplane.PublishInput{
		Topic:   aws.String(topic),
		Payload: payload,
		Qos:     1,
	})
	if err != nil {
		return fmt.Errorf("publish mqtt message: %w", err)
	}

	return nil
}

// PublishWithQoS sends a raw payload using the specified QoS.
//
// QoS:
//
//	0 - At most once
//	1 - At least once
func (s *Service) PublishWithQoS(
	ctx context.Context,
	topic string,
	payload []byte,
	qos int32,
) error {
	if topic == "" {
		return fmt.Errorf("topic is empty")
	}

	if qos != 0 && qos != 1 {
		return fmt.Errorf("invalid qos: %d", qos)
	}

	_, err := s.iotDataPlane.Publish(ctx, &iotdataplane.PublishInput{
		Topic:   aws.String(topic),
		Payload: payload,
		Qos:     qos,
	})
	if err != nil {
		return fmt.Errorf("publish mqtt message: %w", err)
	}

	return nil
}

// PublishRetained publishes a retained MQTT message.
//
// Retained messages are immediately delivered to future subscribers.
//
// This is useful for:
//
//   - printer online status
//   - printer capabilities
//   - firmware version
//   - current print progress snapshot
func (s *Service) PublishRetained(
	ctx context.Context,
	topic string,
	payload []byte,
) error {
	if topic == "" {
		return fmt.Errorf("topic is empty")
	}

	_, err := s.iotDataPlane.Publish(ctx, &iotdataplane.PublishInput{
		Topic:   aws.String(topic),
		Payload: payload,
		Qos:     1,
		Retain:  *aws.Bool(true),
	})
	if err != nil {
		return fmt.Errorf("publish retained mqtt message: %w", err)
	}

	return nil
}

// PublishJSON marshals a Go object into JSON and publishes it.
//
// Example:
//
//	err := svc.PublishJSON(ctx,
//	    "printer/123/status",
//	    status,
//	)
func (s *Service) PublishJSON(
	ctx context.Context,
	topic string,
	v any,
) error {
	payload, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	return s.Publish(ctx, topic, payload)
}
