// Stub pubsub implementation for testing.
package pubsubtest

import (
	"context"
	"sync"
)

// Message is a stub implementation of pubsub.Message for testing
type Message struct {
	Raw        []byte
	AckCalled  bool
	NackCalled bool
	Wg         sync.WaitGroup
}

func (m *Message) Data() []byte {
	return m.Raw
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
	Called bool
	Err    error
}

func (p *Publisher) Publish(_ context.Context, data []byte) error {
	p.Called = true
	p.Data = data
	return p.Err
}
