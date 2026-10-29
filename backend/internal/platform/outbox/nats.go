package outbox

import (
	"context"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
)

// NATSPublisher publishes envelopes on core NATS (JetStream is deferred, ADR-006). The outbox, not NATS,
// is the durable source: an event whose publish cannot be confirmed with a server round-trip stays pending.
type NATSPublisher struct {
	Conn *nats.Conn
}

// Publish sends the envelope and flushes, so a successful return means the server received it.
func (p *NATSPublisher) Publish(ctx context.Context, subject string, env Envelope, body []byte) error {
	msg := &nats.Msg{Subject: subject, Data: body, Header: nats.Header{}}
	msg.Header.Set("Nats-Msg-Id", env.ID) // consumers and a future JetStream use this for de-duplication
	msg.Header.Set("Content-Type", "application/cloudevents+json")
	if env.Traceparent != "" {
		msg.Header.Set("traceparent", env.Traceparent)
	}
	if err := p.Conn.PublishMsg(msg); err != nil {
		return fmt.Errorf("nats publish: %w", err)
	}
	fctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := p.Conn.FlushWithContext(fctx); err != nil {
		return fmt.Errorf("nats flush: %w", err)
	}
	return nil
}
