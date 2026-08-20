package iot

import (
	"encoding/json"
	"testing"
)

func newTestService() *Service {
	return &Service{
		cfg: &Config{
			Region:    "ap-southeast-1",
			AccountID: "123456789012",
		},
	}
}

func TestNewClientID(t *testing.T) {
	id1 := NewClientID()
	id2 := NewClientID()

	if id1 == "" {
		t.Fatal("client id should not be empty")
	}

	if id1 == id2 {
		t.Fatal("client ids should be unique")
	}

	if len(id1) <= len("app_") {
		t.Fatalf("unexpected client id: %s", id1)
	}

	if id1[:4] != "app_" {
		t.Fatalf("unexpected prefix: %s", id1)
	}
}

func TestClientARN(t *testing.T) {
	svc := newTestService()

	got := svc.clientARN("client-001")

	want := "arn:aws:iot:ap-southeast-1:123456789012:client/client-001"

	if got != want {
		t.Fatalf("got %s, want %s", got, want)
	}
}

func TestTopicARN(t *testing.T) {
	svc := newTestService()

	got := svc.topicARN("printer/001/status")

	want := "arn:aws:iot:ap-southeast-1:123456789012:topic/printer/001/status"

	if got != want {
		t.Fatalf("got %s, want %s", got, want)
	}
}

func TestTopicFilterARN(t *testing.T) {
	svc := newTestService()

	got := svc.topicFilterARN("printer/+/status")

	want := "arn:aws:iot:ap-southeast-1:123456789012:topicfilter/printer/+/status"

	if got != want {
		t.Fatalf("got %s, want %s", got, want)
	}
}

func TestTopicARNs(t *testing.T) {
	svc := newTestService()

	got := svc.topicARNs([]string{
		"a",
		"b",
		"c",
	})

	want := []string{
		"arn:aws:iot:ap-southeast-1:123456789012:topic/a",
		"arn:aws:iot:ap-southeast-1:123456789012:topic/b",
		"arn:aws:iot:ap-southeast-1:123456789012:topic/c",
	}

	if len(got) != len(want) {
		t.Fatalf("unexpected length")
	}

	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("index %d got %s want %s", i, got[i], want[i])
		}
	}
}

func TestTopicFilterARNs(t *testing.T) {
	svc := newTestService()

	got := svc.topicFilterARNs([]string{
		"a",
		"b",
	})

	want := []string{
		"arn:aws:iot:ap-southeast-1:123456789012:topicfilter/a",
		"arn:aws:iot:ap-southeast-1:123456789012:topicfilter/b",
	}

	if len(got) != len(want) {
		t.Fatalf("unexpected length")
	}

	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("index %d got %s want %s", i, got[i], want[i])
		}
	}
}

func TestMarshalJSON(t *testing.T) {
	type payload struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	v := payload{
		Name: "printer",
		Age:  1,
	}

	s, err := marshalJSON(v)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var out payload

	if err := json.Unmarshal([]byte(s), &out); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if out != v {
		t.Fatalf("unexpected value")
	}
}

func TestTopicARNsEmpty(t *testing.T) {
	svc := newTestService()

	got := svc.topicARNs(nil)

	if len(got) != 0 {
		t.Fatal("expected empty slice")
	}
}

func TestTopicFilterARNsEmpty(t *testing.T) {
	svc := newTestService()

	got := svc.topicFilterARNs(nil)

	if len(got) != 0 {
		t.Fatal("expected empty slice")
	}
}
