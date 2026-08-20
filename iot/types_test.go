package iot

import (
	"encoding/json"
	"testing"
	"time"
)

func TestCredentialsJSON(t *testing.T) {
	input := Credentials{
		AccessKeyID: "AKIA_TEST",

		SecretAccessKey: "SECRET_TEST",

		SessionToken: "TOKEN_TEST",

		Expiration: time.Date(
			2026,
			1,
			1,
			0,
			0,
			0,
			0,
			time.UTC,
		),
	}

	data, err := json.Marshal(input)
	if err != nil {
		t.Fatalf(
			"marshal error:%v",
			err,
		)
	}

	var output Credentials

	err = json.Unmarshal(
		data,
		&output,
	)
	if err != nil {
		t.Fatalf(
			"unmarshal error:%v",
		)
	}

	if output.AccessKeyID != input.AccessKeyID {
		t.Fatal(
			"access key mismatch",
		)
	}

	if output.SecretAccessKey != input.SecretAccessKey {
		t.Fatal(
			"secret mismatch",
		)
	}

	if output.SessionToken != input.SessionToken {
		t.Fatal(
			"token mismatch",
		)
	}
}

func TestTopicPermissionJSON(t *testing.T) {
	input := TopicPermission{
		Publish: []string{
			"printer/device001/command",
		},

		Subscribe: []string{
			"printer/device001/status",
		},
	}

	data, err := json.Marshal(input)
	if err != nil {
		t.Fatal(err)
	}

	var output TopicPermission

	err = json.Unmarshal(
		data,
		&output,
	)
	if err != nil {
		t.Fatal(err)
	}

	if len(output.Publish) != 1 {
		t.Fatal(
			"publish length mismatch",
		)
	}

	if output.Publish[0] != "printer/device001/command" {
		t.Fatal(
			"publish topic mismatch",
		)
	}

	if len(output.Subscribe) != 1 {
		t.Fatal(
			"subscribe length mismatch",
		)
	}
}

func TestTopicPermissionEmpty(t *testing.T) {
	p := TopicPermission{}

	if len(p.Publish) != 0 {
		t.Fatal(
			"publish should be empty",
		)
	}

	if len(p.Subscribe) != 0 {
		t.Fatal(
			"subscribe should be empty",
		)
	}
}

func TestConnectInfoJSON(t *testing.T) {
	input := ConnectInfo{
		ClientID: "app-user001",

		Endpoint: "xxxxx.iot.amazonaws.com",

		Region: "ap-southeast-1",

		Credentials: Credentials{
			AccessKeyID: "AKIA_TEST",

			SecretAccessKey: "SECRET",

			SessionToken: "TOKEN",
		},
	}

	data, err := json.Marshal(input)
	if err != nil {
		t.Fatal(err)
	}

	var output ConnectInfo

	err = json.Unmarshal(
		data,
		&output,
	)
	if err != nil {
		t.Fatal(err)
	}

	if output.ClientID != input.ClientID {
		t.Fatal(
			"client id mismatch",
		)
	}

	if output.Endpoint != input.Endpoint {
		t.Fatal(
			"endpoint mismatch",
		)
	}

	if output.Credentials.AccessKeyID != "AKIA_TEST" {
		t.Fatal(
			"credential mismatch",
		)
	}
}

func TestCredentialsZeroValue(t *testing.T) {
	var c Credentials

	if c.AccessKeyID != "" {
		t.Fatal(
			"unexpected access key",
		)
	}

	if c.SecretAccessKey != "" {
		t.Fatal(
			"unexpected secret",
		)
	}

	if c.SessionToken != "" {
		t.Fatal(
			"unexpected token",
		)
	}
}

func TestTopicPermissionWithMultipleTopics(t *testing.T) {
	p := TopicPermission{
		Publish: []string{
			"printer/a/cmd",

			"printer/b/cmd",

			"printer/c/cmd",
		},

		Subscribe: []string{
			"printer/a/status",

			"printer/b/status",
		},
	}

	if len(p.Publish) != 3 {
		t.Fatalf(
			"publish count=%d",
			len(p.Publish),
		)
	}

	if len(p.Subscribe) != 2 {
		t.Fatalf(
			"subscribe count=%d",
			len(p.Subscribe),
		)
	}
}
