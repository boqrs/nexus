package iot

// policyDocument represents an IAM policy document used as an
// inline session policy for STS AssumeRole.
//
// It is intentionally kept private because callers should only
// provide TopicPermission instead of constructing IAM policies.
type policyDocument struct {
	Version   string      `json:"Version"`
	Statement []statement `json:"Statement"`
}

// statement represents a single IAM policy statement.
type statement struct {
	Effect    string `json:"Effect"`
	Action    any    `json:"Action"`
	Resource  any    `json:"Resource,omitempty"`
	Condition any    `json:"Condition,omitempty"`
}

// generateScopedPolicy creates an inline IAM policy that grants
// MQTT permissions for a specific client.
//
// The generated policy includes:
//
//   - iot:Connect
//   - iot:Publish
//   - iot:Subscribe
//   - iot:Receive
//
// Receive permissions are automatically derived from Subscribe
// permissions because AWS IoT requires both.
func (s *Service) generateScopedPolicy(
	clientID string,
	permission TopicPermission,
) (string, error) {
	doc := policyDocument{
		Version: "2012-10-17",
		Statement: []statement{
			s.buildConnectStatement(clientID),
		},
	}

	if len(permission.Publish) > 0 {
		doc.Statement = append(
			doc.Statement,
			s.buildPublishStatement(permission.Publish),
		)
	}

	if len(permission.Subscribe) > 0 {

		doc.Statement = append(
			doc.Statement,
			s.buildSubscribeStatement(permission.Subscribe),
		)

		doc.Statement = append(
			doc.Statement,
			s.buildReceiveStatement(permission.Subscribe),
		)
	}

	return marshalJSON(doc)
}

// buildConnectStatement creates the Connect permission.
func (s *Service) buildConnectStatement(
	clientID string,
) statement {
	return statement{
		Effect:   "Allow",
		Action:   "iot:Connect",
		Resource: s.clientARN(clientID),

		Condition: map[string]any{
			"StringEquals": map[string]string{
				"iot:ClientId": clientID,
			},
		},
	}
}

// buildPublishStatement creates the Publish permission.
func (s *Service) buildPublishStatement(
	topics []string,
) statement {
	return statement{
		Effect:   "Allow",
		Action:   "iot:Publish",
		Resource: s.topicARNs(topics),
	}
}

// buildSubscribeStatement creates the Subscribe permission.
//
// AWS IoT requires TopicFilter ARN here.
func (s *Service) buildSubscribeStatement(
	topics []string,
) statement {
	return statement{
		Effect:   "Allow",
		Action:   "iot:Subscribe",
		Resource: s.topicFilterARNs(topics),
	}
}

// buildReceiveStatement creates the Receive permission.
//
// AWS IoT requires Topic ARN here.
func (s *Service) buildReceiveStatement(
	topics []string,
) statement {
	return statement{
		Effect:   "Allow",
		Action:   "iot:Receive",
		Resource: s.topicARNs(topics),
	}
}
