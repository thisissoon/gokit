// Google Cloud PubSub implemenation
package gcloud

import (
	"context"
	"os"

	"cloud.google.com/go/pubsub"
	"github.com/rs/zerolog"
)

// Message implements pubsub.Message for Google Cloud Pubsub
type Message struct {
	*pubsub.Message
}

// Data returns the message data
func (m *Message) Data() []byte {
	return m.Message.Data
}

// Gcloud is an implementation of Publisher/Subscriber for Google Cloud Pubsub
type Gcloud struct {
	subName string
	topic   *pubsub.Topic
	client  *pubsub.Client
	log     zerolog.Logger
}

// Option configures a Gcloud instance
type Option func(*Gcloud)

// WithSubName returns an Option to configure subscription name
func WithSubName(subName string) Option {
	return func(p *Gcloud) {
		p.subName = subName
	}
}

// WithLogger returns an Option to configure the logger
func WithLogger(log zerolog.Logger) Option {
	return func(p *Gcloud) {
		p.log = log.With().
			Str("pkg", "pubsub").
			Str("topic", p.topic.String()).
			Logger()
	}
}

// New sets up a Gcloud instance for Publish/Subscribing to a
// Google Cloud Pubsub topic
func New(ctx context.Context, topic string, client *pubsub.Client, opts ...Option) (*Gcloud, error) {
	log := zerolog.New(os.Stdout).With().
		Str("pkg", "pubsub").
		Str("topic", topic).
		Logger()
	p := &Gcloud{
		subName: "kit",
		client:  client,
		topic:   client.Topic(topic),
		log:     log,
	}
	for _, opt := range opts {
		opt(p)
	}
	return p, nil
}

// Publish implements the Publisher interface for publishing a message
// on a Google Cloud Pubsub topic.
func (p *Gcloud) Publish(ctx context.Context, data []byte) error {
	p.log.Debug().Msg("publishing message")
	p.topic.Publish(ctx, &pubsub.Message{
		Data: data,
	})
	return nil
}

// Closes the underlying topic resources
func (p *Gcloud) Close() {
	p.topic.Stop()
}

// Subscribe implements the Subscriber interface for subscribing to a
// Google Cloud Pubsub topic.
func (p *Gcloud) Subscribe(ctx context.Context) (<-chan Message, error) {
	c := make(chan Message)
	sub := p.client.Subscription(p.subName)
	log := p.log.With().Str("subscription", sub.ID()).Logger()
	go func() {
		log.Debug().Msg("receiving from subscription")
		err := sub.Receive(ctx, func(ctx context.Context, m *pubsub.Message) {
			c <- Message{m}
		})
		if err != nil {
			log.Error().Err(err).Msg("err consuming pubsub message")
		}
	}()
	return c, nil
}
