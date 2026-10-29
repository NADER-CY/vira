// Package httpx contains the HTTP foundations shared by all runtimes: a route-recording router
// (used by contract tests), RFC 9457 problem responses, JSON helpers and common middleware.
package httpx

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"runtime/debug"
	"sort"
	"strings"
	"time"

	"github.com/NADER-CY/vira/backend/internal/platform/apperr"
	"github.com/NADER-CY/vira/backend/internal/platform/logx"
)

// APIVersion is the current major version mounted at /v1 (doc 07 §1: URI major versioning).
const APIVersion = "1"

// Middleware wraps a handler.
type Middleware func(http.Handler) http.Handler

// Chain applies middleware so that the first argument is the outermost.
func Chain(h http.Handler, mw ...Middleware) http.Handler {
	for i := len(mw) - 1; i >= 0; i-- {
		h = mw[i](h)
	}
	return h
}

// Route is a registered route (method + path pattern), used to check the router against the OpenAPI contract.
type Route struct {
	Method string
	Path   string
}

// Router is a thin wrapper over http.ServeMux that records routes.
type Router struct {
	mux    *http.ServeMux
	routes []Route
}

// NewRouter creates a router whose unknown paths answer with problem+json 404.
func NewRouter() *Router {
	return &Router{mux: http.NewServeMux()}
}

// Handle registers h for method and path (Go 1.22+ pattern syntax, e.g. "/v1/admin/flags/{key}").
func (r *Router) Handle(method, path string, h http.Handler, mw ...Middleware) {
	r.routes = append(r.routes, Route{Method: method, Path: path})
	r.mux.Handle(method+" "+path, Chain(h, mw...))
}

// HandleFunc is Handle for functions.
func (r *Router) HandleFunc(method, path string, h http.HandlerFunc, mw ...Middleware) {
	r.Handle(method, path, h, mw...)
}

// Routes returns the registered routes sorted by path then method.
func (r *Router) Routes() []Route {
	out := append([]Route(nil), r.routes...)
	sort.Slice(out, func(i, j int) bool {
		if out[i].Path != out[j].Path {
			return out[i].Path < out[j].Path
		}
		return out[i].Method < out[j].Method
	})
	return out
}

// ServeHTTP implements http.Handler. ServeMux answers unmatched requests with plain-text 404/405; those are
// converted into problem+json here (the Allow header of a 405 is kept).
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	h, pattern := r.mux.Handler(req)
	if pattern != "" {
		r.mux.ServeHTTP(w, req)
		return
	}
	probe := &headerProbe{header: http.Header{}}
	h.ServeHTTP(probe, req)
	if allow := probe.header.Get("Allow"); allow != "" {
		w.Header().Set("Allow", allow)
	}
	if probe.status == http.StatusMethodNotAllowed {
		WriteError(w, req, apperr.New(apperr.MethodNotAllowed, ""))
		return
	}
	WriteError(w, req, apperr.New(apperr.NotFound, "no route"))
}

// headerProbe records the status and headers of ServeMux's built-in 404/405 handlers without writing them.
type headerProbe struct {
	header http.Header
	status int
}

func (p *headerProbe) Header() http.Header         { return p.header }
func (p *headerProbe) Write(b []byte) (int, error) { return len(b), nil }
func (p *headerProbe) WriteHeader(code int)        { p.status = code }

// Problem is an RFC 9457 problem document (doc 07 §1).
type Problem struct {
	Type     string              `json:"type"`
	Title    string              `json:"title"`
	Status   int                 `json:"status"`
	Code     apperr.Code         `json:"code"`
	Detail   string              `json:"detail,omitempty"`
	TraceID  string              `json:"trace_id,omitempty"`
	Errors   []apperr.FieldError `json:"errors,omitempty"`
	Instance string              `json:"instance,omitempty"`
	Extra    map[string]any      `json:"-"`
}

// MarshalJSON flattens extension members into the problem object.
func (p Problem) MarshalJSON() ([]byte, error) {
	type plain Problem
	base, err := json.Marshal(plain(p))
	if err != nil || len(p.Extra) == 0 {
		return base, err
	}
	m := map[string]any{}
	if err := json.Unmarshal(base, &m); err != nil {
		return nil, err
	}
	for k, v := range p.Extra {
		if _, taken := m[k]; !taken {
			m[k] = v
		}
	}
	return json.Marshal(m)
}

// WriteError renders err as problem+json. Internal errors are logged with their cause; clients only see the code.
func WriteError(w http.ResponseWriter, r *http.Request, err error) {
	e := apperr.As(err)
	status := e.Status()
	if status >= 500 {
		logx.From(r.Context()).Error("request failed", "code", e.Code, "err", err)
	}
	detail := e.Detail
	if status >= 500 {
		detail = "" // never leak internals
	}
	p := Problem{
		Type:    "urn:vira:error:" + string(e.Code),
		Title:   apperr.Title(e.Code),
		Status:  status,
		Code:    e.Code,
		Detail:  detail,
		TraceID: TraceID(r.Context()),
		Errors:  e.Fields,
		Extra:   e.Extra,
	}
	w.Header().Set("Content-Type", "application/problem+json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(p)
}

// WriteJSON writes v as JSON with the given status.
func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// MaxBodyBytes bounds request bodies read by DecodeJSON.
const MaxBodyBytes = 1 << 20

// DecodeJSON strictly decodes a JSON request body into v (unknown fields are rejected).
func DecodeJSON(r *http.Request, v any) error {
	if ct := r.Header.Get("Content-Type"); ct != "" {
		mt, _, err := mime.ParseMediaType(ct)
		if err != nil || mt != "application/json" {
			return apperr.Validation(apperr.FieldError{Field: "Content-Type", Code: "unsupported_media_type"})
		}
	}
	dec := json.NewDecoder(io.LimitReader(r.Body, MaxBodyBytes))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return apperr.Validation(apperr.FieldError{Field: "body", Code: "invalid_json", Detail: err.Error()})
	}
	if dec.More() {
		return apperr.Validation(apperr.FieldError{Field: "body", Code: "invalid_json", Detail: "trailing data"})
	}
	return nil
}

type traceKey struct{}

// TraceID returns the request's trace id (from W3C traceparent or generated).
func TraceID(ctx context.Context) string {
	s, _ := ctx.Value(traceKey{}).(string)
	return s
}

// WithTraceID returns a context carrying the trace id (tests and background work).
func WithTraceID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, traceKey{}, id)
}

func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// parseTraceparent validates a W3C traceparent and returns its trace id.
func parseTraceparent(tp string) (string, bool) {
	parts := strings.Split(tp, "-")
	if len(parts) != 4 || len(parts[0]) != 2 || len(parts[1]) != 32 || len(parts[2]) != 16 || len(parts[3]) != 2 {
		return "", false
	}
	if _, err := hex.DecodeString(parts[1]); err != nil || parts[1] == strings.Repeat("0", 32) {
		return "", false
	}
	return parts[1], true
}

// RequestContext assigns a trace id (accepting a valid incoming traceparent), echoes traceparent and
// X-Request-Id, and attaches a request-scoped logger.
func RequestContext(base *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			traceID, ok := parseTraceparent(r.Header.Get("traceparent"))
			if !ok {
				traceID = randomHex(16)
			}
			reqID := r.Header.Get("X-Request-Id")
			if reqID == "" || len(reqID) > 128 {
				reqID = randomHex(8)
			}
			w.Header().Set("traceparent", fmt.Sprintf("00-%s-%s-01", traceID, randomHex(8)))
			w.Header().Set("X-Request-Id", reqID)
			ctx := WithTraceID(r.Context(), traceID)
			ctx = logx.With(ctx, base.With("trace_id", traceID, "request_id", reqID))
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// statusRecorder captures the response status for access logs.
type statusRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (s *statusRecorder) WriteHeader(code int) {
	if s.status == 0 {
		s.status = code
	}
	s.ResponseWriter.WriteHeader(code)
}

func (s *statusRecorder) Write(b []byte) (int, error) {
	if s.status == 0 {
		s.status = http.StatusOK
	}
	n, err := s.ResponseWriter.Write(b)
	s.bytes += n
	return n, err
}

// Unwrap lets http.ResponseController reach the underlying writer (needed for WebSocket hijacking).
func (s *statusRecorder) Unwrap() http.ResponseWriter { return s.ResponseWriter }

// AccessLog logs one structured line per request (no query strings, no bodies, no headers with credentials).
func AccessLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w}
		next.ServeHTTP(rec, r)
		if rec.status == 0 {
			rec.status = http.StatusOK
		}
		logx.From(r.Context()).Info("http",
			"method", r.Method, "path", r.URL.Path, "status", rec.status,
			"bytes", rec.bytes, "duration_ms", time.Since(start).Milliseconds())
	})
}

// Recover converts panics into 500 problem responses.
func Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if v := recover(); v != nil {
				if v == http.ErrAbortHandler {
					panic(v)
				}
				logx.From(r.Context()).Error("panic", "value", fmt.Sprint(v), "stack", string(debug.Stack()))
				WriteError(w, r, apperr.Wrap(apperr.Internal, errors.New("panic")))
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// SecurityHeaders sets conservative headers for JSON APIs.
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
		h.Set("API-Version", APIVersion)
		next.ServeHTTP(w, r)
	})
}

// CORS allows credentialed requests from an explicit origin allow-list only (admin and seller web).
func CORS(allowed []string) Middleware {
	set := map[string]bool{}
	for _, o := range allowed {
		set[o] = true
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin != "" && set[origin] {
				h := w.Header()
				h.Set("Access-Control-Allow-Origin", origin)
				h.Set("Access-Control-Allow-Credentials", "true")
				h.Add("Vary", "Origin")
				if r.Method == http.MethodOptions {
					h.Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE")
					h.Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Idempotency-Key, If-Match, X-Request-Id, X-Store-Id, traceparent")
					h.Set("Access-Control-Max-Age", "600")
					w.WriteHeader(http.StatusNoContent)
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}
