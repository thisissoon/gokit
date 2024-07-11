// Google Cloud PubSub implemenation
package gcloud

import (
	"context"
	"os"

	"cloud.google.com/go/pubsub"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

// Message implements pubsub.Message for Google Cloud Pubsub
type Message struct {
	*pubsub.Message
	propagator propagation.TextMapPropagator
}

// Data returns the message data
func (m *Message) Data() []byte {
	return m.Message.Data
}

// Returns a new context that is enriched by the propagator passed during
// the pubsub client's construction.
func (m *Message) EnrichContext(ctx context.Context) context.Context {
	if m.propagator == nil { // This generally shouldn't happen, but is here as a safeguard.
		return ctx
	}

	return m.propagator.Extract(ctx, propagation.MapCarrier(m.Attributes))
}

// Gcloud is an implementation of Publisher/Subscriber for Google Cloud Pubsub
type Gcloud struct {
	subName    string
	topic      *pubsub.Topic
	client     *pubsub.Client
	log        zerolog.Logger
	propagator propagation.TextMapPropagator
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

func WithPropagator(propagator propagation.TextMapPropagator) Option {
	return func(p *Gcloud) {
		p.propagator = propagator
	}
}

// New sets up a Gcloud instance for Publish/Subscribing to a
// Google Cloud Pubsub topic.
//
// Please note that if `WithPropagator` is not used then the default
// OTEL propagator will be used. This mean you'll need to initialise OTEL before this client
// if you rely on this behaviour.
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
	if p.propagator == nil {
		p.propagator = otel.GetTextMapPropagator()
	}
	return p, nil
}

// Publish implements the Publisher interface for publishing a message
// on a Google Cloud Pubsub topic.
//
// The client's propagator will be used to inject attributes into the message.
func (p *Gcloud) Publish(ctx context.Context, data []byte) error {
	p.log.Debug().Msg("publishing message")

	attributes := make(map[string]string)
	p.propagator.Inject(ctx, propagation.MapCarrier(attributes))

	p.topic.Publish(ctx, &pubsub.Message{
		Data:       data,
		Attributes: attributes,
	})
	return nil
}

// PublishUntilComplete is similar to Publish, but is a blocking call as it uses `.Get()`,
// it will also return any error that occurs
func (p *Gcloud) PublishUntilComplete(ctx context.Context, data []byte) error {
	p.log.Debug().Msg("publishing message until complete")

	attributes := make(map[string]string)
	p.propagator.Inject(ctx, propagation.MapCarrier(attributes))

	_, err := p.topic.Publish(ctx, &pubsub.Message{
		Data:       data,
		Attributes: attributes,
	}).Get(ctx)
	return err
}

// Closes the underlying topic resources
func (p *Gcloud) Close() {
	p.topic.Stop()
}

// Subscribe implements the Subscriber interface for subscribing to a
// Google Cloud Pubsub topic.
//
// The client's propagator will be used to extract attributes from each message,
// which the callback can make use of by calling `Message.EnrichContext`.
func (p *Gcloud) Subscribe(ctx context.Context) (<-chan Message, error) {
	c := make(chan Message)
	sub := p.client.Subscription(p.subName)
	log := p.log.With().Str("subscription", sub.ID()).Logger()
	go func() {
		log.Debug().Msg("receiving from subscription")
		err := sub.Receive(ctx, func(ctx context.Context, m *pubsub.Message) {
			c <- Message{m, p.propagator}
		})
		if err != nil {
			log.Error().Err(err).Msg("err consuming pubsub message")
		}
	}()
	return c, nil
}
