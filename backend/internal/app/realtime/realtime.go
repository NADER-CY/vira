// Package realtime implements the WebSocket gateway skeleton (doc 07 §3): authenticated hello/ready
// handshake, heartbeats and ping/pong. Message, notification and presence fan-out over NATS arrive with
// messaging in Phase 5; the protocol envelope and authentication are fixed now.
package realtime

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/coder/websocket"

	"github.com/NADER-CY/vira/backend/internal/platform/auth"
	"github.com/NADER-CY/vira/backend/internal/platform/health"
	"github.com/NADER-CY/vira/backend/internal/platform/httpx"
	"github.com/NADER-CY/vira/backend/internal/platform/ids"
)

// Heartbeat is the interval clients must ping within (doc 07 §3: heartbeat_ms 25000).
const Heartbeat = 25 * time.Second

// HelloTimeout bounds the unauthenticated phase of a connection.
const HelloTimeout = 10 * time.Second

// Deps are the gateway's dependencies.
type Deps struct {
	Verifier       *auth.Verifier
	Health         *health.Handler
	Log            *slog.Logger
	AllowedOrigins []string // Origin patterns accepted for browser clients (seller web)
}

// Frame is the protocol envelope.
type Frame struct {
	Op          string `json:"op"`
	Token       string `json:"token,omitempty"`
	Resume      string `json:"resume,omitempty"`
	Session     string `json:"session,omitempty"`
	HeartbeatMS int64  `json:"heartbeat_ms,omitempty"`
	Code        string `json:"code,omitempty"`
}

// New builds the realtime router.
func New(d Deps) *httpx.Router {
	r := httpx.NewRouter()
	d.Health.Register(r)
	r.HandleFunc(http.MethodGet, "/v1/ws", func(w http.ResponseWriter, req *http.Request) { serve(d, w, req) })
	return r
}

// Handler wraps the router with the standard middleware stack (no SecurityHeaders: WebSocket upgrade).
func Handler(d Deps, r *httpx.Router) http.Handler {
	return httpx.Chain(r, httpx.RequestContext(d.Log), httpx.Recover, httpx.AccessLog)
}

func serve(d Deps, w http.ResponseWriter, r *http.Request) {
	c, err := websocket.Accept(w, r, &websocket.AcceptOptions{OriginPatterns: d.AllowedOrigins})
	if err != nil {
		return // Accept already wrote the HTTP error
	}
	defer c.CloseNow() //nolint:errcheck
	c.SetReadLimit(64 << 10)
	ctx := r.Context()

	// 1. hello with a valid access token, within HelloTimeout.
	hctx, cancel := context.WithTimeout(ctx, HelloTimeout)
	hello, err := read(hctx, c)
	cancel()
	if err != nil || hello.Op != "hello" {
		_ = c.Close(websocket.StatusPolicyViolation, "hello required")
		return
	}
	claims, err := d.Verifier.Verify(hello.Token)
	if err != nil {
		_ = write(ctx, c, Frame{Op: "error", Code: "unauthenticated"})
		_ = c.Close(4401, "unauthenticated")
		return
	}
	session := ids.New()
	if err := write(ctx, c, Frame{Op: "ready", Session: session, HeartbeatMS: Heartbeat.Milliseconds()}); err != nil {
		return
	}
	d.Log.Info("realtime session", "session", session, "subject", claims.Subject)

	// 2. Serve until the token expires, the client goes silent, or the connection closes.
	expires := time.Unix(claims.Expires, 0)
	for {
		deadline := time.Now().Add(2 * Heartbeat)
		if expires.Before(deadline) {
			deadline = expires
		}
		rctx, cancel := context.WithDeadline(ctx, deadline)
		f, err := read(rctx, c)
		cancel()
		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				if time.Now().After(expires) {
					_ = c.Close(4401, "token expired")
				} else {
					_ = c.Close(websocket.StatusPolicyViolation, "heartbeat timeout")
				}
			}
			return
		}
		switch f.Op {
		case "ping":
			if err := write(ctx, c, Frame{Op: "pong"}); err != nil {
				return
			}
		default:
			if err := write(ctx, c, Frame{Op: "error", Code: "unsupported_op"}); err != nil {
				return
			}
		}
	}
}

func read(ctx context.Context, c *websocket.Conn) (Frame, error) {
	typ, b, err := c.Read(ctx)
	if err != nil {
		return Frame{}, err
	}
	if typ != websocket.MessageText {
		return Frame{}, errors.New("binary frames not supported")
	}
	var f Frame
	if err := json.Unmarshal(b, &f); err != nil {
		return Frame{}, err
	}
	return f, nil
}

func write(ctx context.Context, c *websocket.Conn, f Frame) error {
	b, _ := json.Marshal(f)
	wctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	return c.Write(wctx, websocket.MessageText, b)
}
