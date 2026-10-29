package realtime

import (
	"context"
	"crypto/ecdsa"
	"io"
	"log/slog"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"

	"github.com/NADER-CY/vira/backend/internal/platform/auth"
	"github.com/NADER-CY/vira/backend/internal/platform/health"
)

func server(t *testing.T) (*httptest.Server, *auth.Signer) {
	k, _ := auth.GenerateKey()
	kid := auth.KeyID(&k.PublicKey)
	s := &auth.Signer{Key: k, KeyID: kid, Issuer: "vira", Audience: "vira-api", TTL: time.Minute}
	v := &auth.Verifier{Keys: map[string]*ecdsa.PublicKey{kid: &k.PublicKey}, Issuer: "vira", Audience: "vira-api"}
	d := Deps{Verifier: v, Health: &health.Handler{Service: "realtime"}, Log: slog.New(slog.NewTextHandler(io.Discard, nil))}
	srv := httptest.NewServer(Handler(d, New(d)))
	t.Cleanup(srv.Close)
	return srv, s
}

func dial(t *testing.T, srv *httptest.Server) *websocket.Conn {
	c, _, err := websocket.Dial(context.Background(), "ws"+strings.TrimPrefix(srv.URL, "http")+"/v1/ws", nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { c.CloseNow() }) //nolint:errcheck
	return c
}

func TestHelloReadyPingPong(t *testing.T) {
	srv, signer := server(t)
	tok, _, _ := signer.Issue("user-1", "user", nil)
	c := dial(t, srv)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := write(ctx, c, Frame{Op: "hello", Token: tok}); err != nil {
		t.Fatal(err)
	}
	f, err := read(ctx, c)
	if err != nil || f.Op != "ready" || f.Session == "" || f.HeartbeatMS != 25000 {
		t.Fatalf("ready: %+v %v", f, err)
	}
	write(ctx, c, Frame{Op: "ping"}) //nolint:errcheck
	if f, err := read(ctx, c); err != nil || f.Op != "pong" {
		t.Fatalf("pong: %+v %v", f, err)
	}
	write(ctx, c, Frame{Op: "send"}) //nolint:errcheck
	if f, err := read(ctx, c); err != nil || f.Op != "error" || f.Code != "unsupported_op" {
		t.Fatalf("unsupported: %+v %v", f, err)
	}
}

func TestInvalidTokenClosesConnection(t *testing.T) {
	srv, _ := server(t)
	c := dial(t, srv)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	write(ctx, c, Frame{Op: "hello", Token: "bad"}) //nolint:errcheck
	if f, err := read(ctx, c); err != nil || f.Op != "error" || f.Code != "unauthenticated" {
		t.Fatalf("got %+v %v", f, err)
	}
	_, err := read(ctx, c)
	if websocket.CloseStatus(err) != 4401 {
		t.Fatalf("close status %v", websocket.CloseStatus(err))
	}
}

func TestFirstFrameMustBeHello(t *testing.T) {
	srv, _ := server(t)
	c := dial(t, srv)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	write(ctx, c, Frame{Op: "ping"}) //nolint:errcheck
	_, err := read(ctx, c)
	if websocket.CloseStatus(err) != websocket.StatusPolicyViolation {
		t.Fatalf("close status %v", websocket.CloseStatus(err))
	}
}
