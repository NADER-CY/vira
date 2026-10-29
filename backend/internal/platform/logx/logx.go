// Package logx configures structured JSON logging (log/slog) for every runtime.
package logx

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strings"
)

type ctxKey struct{}

// New returns a JSON logger tagged with service and environment.
func New(service, env, level string) *slog.Logger { return NewWriter(os.Stdout, service, env, level) }

// NewWriter is New with an explicit writer (tests).
func NewWriter(w io.Writer, service, env, level string) *slog.Logger {
	var lvl slog.Level
	switch strings.ToLower(level) {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}
	h := slog.NewJSONHandler(w, &slog.HandlerOptions{Level: lvl, ReplaceAttr: redact})
	return slog.New(h).With("service", service, "env", env)
}

// sensitive attribute keys are never written to logs (doc 08: no secrets or credentials in logs).
var sensitive = map[string]bool{
	"password": true, "token": true, "access_token": true, "refresh_token": true, "authorization": true,
	"cookie": true, "set-cookie": true, "secret": true, "totp": true, "otp": true, "code": false,
}

func redact(_ []string, a slog.Attr) slog.Attr {
	if sensitive[strings.ToLower(a.Key)] {
		return slog.String(a.Key, "[REDACTED]")
	}
	return a
}

// With stores a logger in the context.
func With(ctx context.Context, l *slog.Logger) context.Context {
	return context.WithValue(ctx, ctxKey{}, l)
}

// From returns the request-scoped logger, or the default logger.
func From(ctx context.Context) *slog.Logger {
	if l, ok := ctx.Value(ctxKey{}).(*slog.Logger); ok {
		return l
	}
	return slog.Default()
}
