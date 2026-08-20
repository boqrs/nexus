package iot

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
)

// NewClientID generates a globally unique MQTT client ID.
func NewClientID() string {
	return "app_" + uuid.NewString()
}

// clientARN returns the ARN of an MQTT client.
func (s *Service) clientARN(clientID string) string {
	return fmt.Sprintf(
		"arn:aws:iot:%s:%s:client/%s",
		s.cfg.Region,
		s.cfg.AccountID,
		clientID,
	)
}

// topicARN returns the ARN of an MQTT topic.
func (s *Service) topicARN(topic string) string {
	return fmt.Sprintf(
		"arn:aws:iot:%s:%s:topic/%s",
		s.cfg.Region,
		s.cfg.AccountID,
		topic,
	)
}

// topicFilterARN returns the ARN of an MQTT topic filter.
func (s *Service) topicFilterARN(topic string) string {
	return fmt.Sprintf(
		"arn:aws:iot:%s:%s:topicfilter/%s",
		s.cfg.Region,
		s.cfg.AccountID,
		topic,
	)
}

// topicARNs converts MQTT topics into topic ARNs.
func (s *Service) topicARNs(topics []string) []string {
	resources := make([]string, 0, len(topics))

	for _, topic := range topics {
		resources = append(resources, s.topicARN(topic))
	}

	return resources
}

// topicFilterARNs converts MQTT topic filters into topic filter ARNs.
func (s *Service) topicFilterARNs(topics []string) []string {
	resources := make([]string, 0, len(topics))

	for _, topic := range topics {
		resources = append(resources, s.topicFilterARN(topic))
	}

	return resources
}

func marshalJSON(v any) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}

	return string(b), nil
}