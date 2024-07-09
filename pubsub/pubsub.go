// Provides a super minimal publish/subscribe interface with backend
// implementations for different providers
package pubsub

import (
	"context"
)

// A Message is a pubsub message to be consumed by a Worker
type Message interface {
	Data() []byte
	Ack()
	Nack()
	EnrichContext(context.Context) context.Context
}

// A Publisher takes some data and publishes it on a topic
type Publisher interface {
	Publish(context.Context, []byte) error
}

// A Subscriber streams a channel of Message
type Subscriber interface {
	Subscribe(context.Context) (<-chan Message, error)
}
