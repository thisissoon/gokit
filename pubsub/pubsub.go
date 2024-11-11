// Provides a super minimal publish/subscribe interface with backend
// implementations for different providers
package pubsub

import (
	"context"
)

// A Message is a pubsub message to be consumed by a Worker
type Message interface {
	Data() []byte
	Attrs() map[string]string
	Ack()
	Nack()
	EnrichContext(context.Context) context.Context
}

// A Publisher takes some data and publishes it on a topic
type Publisher interface {
	Publish(context.Context, []byte, map[string]string) error
}

// A CompletePublisher takes some data and publishes it on a topic
// and returns errors, if any
type CompletePublisher interface {
	PublishUntilComplete(context.Context, []byte, map[string]string) error
}

// A Subscriber streams a channel of Message
type Subscriber interface {
	Subscribe(context.Context) (<-chan Message, error)
}
