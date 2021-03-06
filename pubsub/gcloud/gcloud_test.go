// +build gcloud

package gcloud_test

import (
	"context"
	"testing"

	"go.soon.build/kit/pubsub/gcloud"

	"cloud.google.com/go/pubsub"
)

func TestPublishSubscribe(t *testing.T) {
	client, err := pubsub.NewClient(context.Background(), "kit-test")
	if err != nil {
		t.Fatal(err)
	}
	p, err := gcloud.New(context.Background(), "test", client)
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	// subscribe
	C, err := p.Subscribe(context.Background())
	if err != nil {
		t.Error(err)
	}
	// publish
	data := []byte(`{"data": "data"}`)
	errC := make(chan error, 1)
	go func() {
		err = p.Publish(context.Background(), data)
		if err != nil {
			errC <- err
		}
		close(errC)
	}()
	// assert publish errs
	err = <-errC
	if err != nil {
		t.Error(err)
	}
	// assert msg
	msg := <-C
	if string(msg.Data()) != string(data) {
		t.Errorf("unexpected msg data; expected %v, got %v", data, msg.Data())
	}
	msg.Ack()
}
