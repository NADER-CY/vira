// Package sms sends one-time codes. Only development senders exist until a real provider is contracted and
// tested (doc 13 dependency register); configuration refuses them in production.
package sms

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// Sender delivers a text message to an Iranian mobile number (09xxxxxxxxx).
type Sender interface {
	Send(ctx context.Context, to, text string) error
}

// Devsim posts messages to the development simulator inbox (cmd/devsim).
type Devsim struct {
	URL    string
	Client *http.Client
}

// Send delivers to the simulator.
func (d *Devsim) Send(ctx context.Context, to, text string) error {
	b, _ := json.Marshal(map[string]string{"to": to, "text": text})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, d.URL+"/sms/v1/send", bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	c := d.Client
	if c == nil {
		c = &http.Client{Timeout: 5 * time.Second}
	}
	resp, err := c.Do(req)
	if err != nil {
		return fmt.Errorf("sms devsim: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("sms devsim: status %d", resp.StatusCode)
	}
	return nil
}

// Message is a captured message (Mock).
type Message struct{ To, Text string }

// Mock captures messages in memory (tests, local runs without the simulator).
type Mock struct {
	mu   sync.Mutex
	msgs []Message
	Fail bool
}

// Send captures the message.
func (m *Mock) Send(_ context.Context, to, text string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.Fail {
		return fmt.Errorf("sms mock: configured to fail")
	}
	m.msgs = append(m.msgs, Message{to, text})
	return nil
}

// Last returns the last message sent to `to`.
func (m *Mock) Last(to string) (Message, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := len(m.msgs) - 1; i >= 0; i-- {
		if m.msgs[i].To == to {
			return m.msgs[i], true
		}
	}
	return Message{}, false
}

// Count returns the number of messages sent to `to`.
func (m *Mock) Count(to string) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	n := 0
	for _, x := range m.msgs {
		if x.To == to {
			n++
		}
	}
	return n
}
