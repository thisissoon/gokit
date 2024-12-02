// Stub pubsub implementation for testing.
package pubsubtest

import (
	"context"
	"sync"
)

// Message is a stub implementation of pubsub.Message for testing
type Message struct {
	Raw        []byte
	Attr       map[string]string
	AckCalled  bool
	NackCalled bool
	Wg         sync.WaitGroup
}

func (m *Message) Data() []byte {
	return m.Raw
}

func (m *Message) Attrs() map[string]string {
	return m.Attr
}

func (m *Message) Ack() {
	m.AckCalled = true
	m.Wg.Done()
}

func (m *Message) Nack() {
	m.NackCalled = true
	m.Wg.Done()
}

// Publisher is a stub implementation of pubsub.Publisher for testing
type Publisher struct {
	Data   []byte
	Attrs  map[string]string
	Called bool
	Err    error
}

func (p *Publisher) Publish(_ context.Context, data []byte, attrs map[string]string) error {
	p.Data = data
	p.Attrs = attrs
	p.Called = true
	return p.Err
}

// Publisher is a stub implementation of pubsub.CompletePublisher for testing
type CompletePublisher struct {
	Data   []byte
	Attrs  map[string]string
	Called bool
	Err    error
}

func (cp *CompletePublisher) PublishUntilComplete(_ context.Context, data []byte, attrs map[string]string) error {
	cp.Data = data
	cp.Attrs = attrs
	cp.Called = true
	return cp.Err
}
