package iot

import (
	"encoding/json"
	"testing"
)

func newPolicyTestService() *Service {
	return &Service{
		cfg: &Config{
			Region:    "ap-southeast-1",
			AccountID: "123456789012",
		},
	}
}

func TestGenerateScopedPolicy(t *testing.T) {
	svc := newPolicyTestService()

	permission := TopicPermission{
		Publish: []string{
			"printer/001/command",
		},
		Subscribe: []string{
			"printer/001/status",
			"printer/001/event",
		},
	}

	policyJSON, err := svc.generateScopedPolicy(
		"client-001",
		permission,
	)
	if err != nil {
		t.Fatalf("generateScopedPolicy() error = %v", err)
	}

	var doc policyDocument

	if err := json.Unmarshal([]byte(policyJSON), &doc); err != nil {
		t.Fatalf("invalid json: %v", err)
	}

	if doc.Version != "2012-10-17" {
		t.Fatalf("unexpected version: %s", doc.Version)
	}

	if len(doc.Statement) != 4 {
		t.Fatalf("expected 4 statements, got %d", len(doc.Statement))
	}

	// Connect
	if doc.Statement[0].Action != "iot:Connect" {
		t.Fatalf("statement[0] should be Connect")
	}

	// Publish
	if doc.Statement[1].Action != "iot:Publish" {
		t.Fatalf("statement[1] should be Publish")
	}

	// Subscribe
	if doc.Statement[2].Action != "iot:Subscribe" {
		t.Fatalf("statement[2] should be Subscribe")
	}

	// Receive
	if doc.Statement[3].Action != "iot:Receive" {
		t.Fatalf("statement[3] should be Receive")
	}
}

func TestGenerateScopedPolicyPublishOnly(t *testing.T) {
	svc := newPolicyTestService()

	permission := TopicPermission{
		Publish: []string{
			"printer/001/command",
		},
	}

	policyJSON, err := svc.generateScopedPolicy(
		"client-001",
		permission,
	)
	if err != nil {
		t.Fatal(err)
	}

	var doc policyDocument

	if err := json.Unmarshal([]byte(policyJSON), &doc); err != nil {
		t.Fatal(err)
	}

	if len(doc.Statement) != 2 {
		t.Fatalf("expected 2 statements, got %d", len(doc.Statement))
	}

	if doc.Statement[1].Action != "iot:Publish" {
		t.Fatal("expected publish statement")
	}
}

func TestGenerateScopedPolicySubscribeOnly(t *testing.T) {
	svc := newPolicyTestService()

	permission := TopicPermission{
		Subscribe: []string{
			"printer/001/status",
		},
	}

	policyJSON, err := svc.generateScopedPolicy(
		"client-001",
		permission,
	)
	if err != nil {
		t.Fatal(err)
	}

	var doc policyDocument

	if err := json.Unmarshal([]byte(policyJSON), &doc); err != nil {
		t.Fatal(err)
	}

	if len(doc.Statement) != 3 {
		t.Fatalf("expected 3 statements, got %d", len(doc.Statement))
	}

	if doc.Statement[1].Action != "iot:Subscribe" {
		t.Fatal("expected subscribe")
	}

	if doc.Statement[2].Action != "iot:Receive" {
		t.Fatal("expected receive")
	}
}

func TestGenerateScopedPolicyConnectOnly(t *testing.T) {
	svc := newPolicyTestService()

	policyJSON, err := svc.generateScopedPolicy(
		"client-001",
		TopicPermission{},
	)
	if err != nil {
		t.Fatal(err)
	}

	var doc policyDocument

	if err := json.Unmarshal([]byte(policyJSON), &doc); err != nil {
		t.Fatal(err)
	}

	if len(doc.Statement) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(doc.Statement))
	}

	if doc.Statement[0].Action != "iot:Connect" {
		t.Fatal("expected connect")
	}
}

func TestBuildConnectStatement(t *testing.T) {
	svc := newPolicyTestService()

	stmt := svc.buildConnectStatement("client-001")

	if stmt.Action != "iot:Connect" {
		t.Fatal("unexpected action")
	}

	if stmt.Effect != "Allow" {
		t.Fatal("unexpected effect")
	}

	resource, ok := stmt.Resource.(string)
	if !ok {
		t.Fatal("resource should be string")
	}

	want := "arn:aws:iot:ap-southeast-1:123456789012:client/client-001"

	if resource != want {
		t.Fatalf("unexpected resource: %s", resource)
	}
}

func TestBuildPublishStatement(t *testing.T) {
	svc := newPolicyTestService()

	stmt := svc.buildPublishStatement([]string{
		"printer/001/command",
	})

	if stmt.Action != "iot:Publish" {
		t.Fatal("unexpected action")
	}

	resources := stmt.Resource.([]string)

	if len(resources) != 1 {
		t.Fatal("unexpected resource count")
	}

	want := "arn:aws:iot:ap-southeast-1:123456789012:topic/printer/001/command"

	if resources[0] != want {
		t.Fatal("unexpected publish arn")
	}
}

func TestBuildSubscribeStatement(t *testing.T) {
	svc := newPolicyTestService()

	stmt := svc.buildSubscribeStatement([]string{
		"printer/+/status",
	})

	if stmt.Action != "iot:Subscribe" {
		t.Fatal("unexpected action")
	}

	resources := stmt.Resource.([]string)

	want := "arn:aws:iot:ap-southeast-1:123456789012:topicfilter/printer/+/status"

	if resources[0] != want {
		t.Fatal("unexpected subscribe arn")
	}
}

func TestBuildReceiveStatement(t *testing.T) {
	svc := newPolicyTestService()

	stmt := svc.buildReceiveStatement([]string{
		"printer/001/status",
	})

	if stmt.Action != "iot:Receive" {
		t.Fatal("unexpected action")
	}

	resources := stmt.Resource.([]string)

	want := "arn:aws:iot:ap-southeast-1:123456789012:topic/printer/001/status"

	if resources[0] != want {
		t.Fatal("unexpected receive arn")
	}
}
