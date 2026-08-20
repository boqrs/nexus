package iot

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/iotdataplane"
)

type fakeIoTDataPlane struct {
	input  *iotdataplane.PublishInput
	output *iotdataplane.PublishOutput
	err    error
}

func (f *fakeIoTDataPlane) Publish(
	ctx context.Context,
	input *iotdataplane.PublishInput,
	optFns ...func(*iotdataplane.Options),
) (*iotdataplane.PublishOutput, error) {
	f.input = input

	if f.err != nil {
		return nil, f.err
	}

	return f.output, nil
}

func TestPublish(t *testing.T) {
	fake := &fakeIoTDataPlane{
		output: &iotdataplane.PublishOutput{},
	}

	svc := &Service{
		iotDataPlane: fake,
	}

	err := svc.Publish(
		context.Background(),
		"printer/001/command",
		[]byte(`{"cmd":"pause"}`),
	)
	if err != nil {
		t.Fatalf("Publish() error = %v", err)
	}

	if fake.input == nil {
		t.Fatal("Publish was not called")
	}

	if *fake.input.Topic != "printer/001/command" {
		t.Fatalf(
			"unexpected topic: %s",
			*fake.input.Topic,
		)
	}

	if string(fake.input.Payload) != `{"cmd":"pause"}` {
		t.Fatalf(
			"unexpected payload: %s",
			string(fake.input.Payload),
		)
	}

	if fake.input.Qos != 1 {
		t.Fatalf(
			"unexpected qos: %d",
			fake.input.Qos,
		)
	}
}

func TestPublishEmptyTopic(t *testing.T) {
	fake := &fakeIoTDataPlane{}

	svc := &Service{
		iotDataPlane: fake,
	}

	err := svc.Publish(
		context.Background(),
		"",
		[]byte("test"),
	)

	if err == nil {
		t.Fatal("expected error")
	}
}

func TestPublishAWSFailure(t *testing.T) {
	fake := &fakeIoTDataPlane{
		err: errors.New("aws publish failed"),
	}

	svc := &Service{
		iotDataPlane: fake,
	}

	err := svc.Publish(
		context.Background(),
		"printer/001/command",
		[]byte("test"),
	)

	if err == nil {
		t.Fatal("expected error")
	}
}

func TestPublishWithQoS(t *testing.T) {
	tests := []struct {
		name string
		qos  int32
	}{
		{
			name: "qos zero",
			qos:  0,
		},
		{
			name: "qos one",
			qos:  1,
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name,
			func(t *testing.T) {
				fake := &fakeIoTDataPlane{}

				svc := &Service{
					iotDataPlane: fake,
				}

				err := svc.PublishWithQoS(
					context.Background(),
					"printer/001/event",
					[]byte("hello"),
					tt.qos,
				)
				if err != nil {
					t.Fatalf(
						"PublishWithQoS error=%v",
						err,
					)
				}

				if fake.input.Qos != tt.qos {
					t.Fatalf(
						"qos=%d want=%d",
						fake.input.Qos,
						tt.qos,
					)
				}
			},
		)
	}
}

func TestPublishWithInvalidQoS(t *testing.T) {
	svc := &Service{
		iotDataPlane: &fakeIoTDataPlane{},
	}

	err := svc.PublishWithQoS(
		context.Background(),
		"printer/001/event",
		[]byte("hello"),
		2,
	)

	if err == nil {
		t.Fatal("expected qos validation error")
	}
}

func TestPublishJSON(t *testing.T) {
	fake := &fakeIoTDataPlane{}

	svc := &Service{
		iotDataPlane: fake,
	}

	type Message struct {
		Command string `json:"command"`
	}

	err := svc.PublishJSON(
		context.Background(),
		"printer/001/command",
		Message{
			Command: "pause",
		},
	)
	if err != nil {
		t.Fatalf(
			"PublishJSON error=%v",
			err,
		)
	}

	if fake.input == nil {
		t.Fatal("PublishJSON did not call Publish")
	}

	want := `{"command":"pause"}`

	if string(fake.input.Payload) != want {
		t.Fatalf(
			"payload=%s want=%s",
			string(fake.input.Payload),
			want,
		)
	}
}
