package idempotency_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/NADER-CY/vira/backend/internal/platform/idempotency"
	"github.com/NADER-CY/vira/backend/internal/platform/testdb"
)

func setup(t *testing.T, handler http.HandlerFunc, required bool) (http.Handler, *testdb.DB) {
	d := testdb.New(t, true)
	m := &idempotency.Middleware{Pool: d.Pool("app_public"), Required: required,
		Principal: func(r *http.Request) string { return r.Header.Get("X-Test-Principal") }}
	return m.Wrap(handler), d
}

func do(h http.Handler, key, principal, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/v1/things", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if key != "" {
		req.Header.Set(idempotency.Header, key)
	}
	req.Header.Set("X-Test-Principal", principal)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestReplayReturnsStoredResponseWithoutReExecuting(t *testing.T) {
	var calls int32
	h, _ := setup(t, func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&calls, 1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"n":` + string(rune('0'+n)) + `}`)) //nolint:errcheck
	}, false)
	first := do(h, "key-00000001", "u1", `{"a":1}`)
	second := do(h, "key-00000001", "u1", `{"a":1}`)
	if calls != 1 {
		t.Fatalf("handler executed %d times", calls)
	}
	if second.Code != http.StatusCreated || second.Body.String() != first.Body.String() {
		t.Fatalf("replay mismatch: %d %q vs %q", second.Code, second.Body.String(), first.Body.String())
	}
	if second.Header().Get("Idempotent-Replayed") != "true" {
		t.Fatal("missing Idempotent-Replayed header")
	}
}

func TestSameKeyDifferentBodyIsConflict(t *testing.T) {
	h, _ := setup(t, func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusCreated) }, false)
	do(h, "key-00000002", "u1", `{"a":1}`)
	rec := do(h, "key-00000002", "u1", `{"a":2}`)
	if rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), `"idempotency_conflict"`) {
		t.Fatalf("got %d %s", rec.Code, rec.Body.String())
	}
}

func TestKeysAreScopedPerPrincipal(t *testing.T) {
	var calls int32
	h, _ := setup(t, func(w http.ResponseWriter, r *http.Request) { atomic.AddInt32(&calls, 1); w.WriteHeader(201) }, false)
	do(h, "key-00000003", "u1", `{}`)
	do(h, "key-00000003", "u2", `{}`)
	if calls != 2 {
		t.Fatalf("calls=%d, keys must not collide across principals", calls)
	}
}

func TestTransientFailureIsNotFinal(t *testing.T) {
	var calls int32
	h, _ := setup(t, func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&calls, 1) == 1 {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		w.WriteHeader(http.StatusCreated)
	}, false)
	if rec := do(h, "key-00000004", "u1", `{}`); rec.Code != http.StatusBadGateway {
		t.Fatalf("first=%d", rec.Code)
	}
	if rec := do(h, "key-00000004", "u1", `{}`); rec.Code != http.StatusCreated || calls != 2 {
		t.Fatalf("retry=%d calls=%d", rec.Code, calls)
	}
}

func TestConcurrentDuplicateGetsInProgress(t *testing.T) {
	release := make(chan struct{})
	started := make(chan struct{})
	var once sync.Once
	h, _ := setup(t, func(w http.ResponseWriter, r *http.Request) {
		once.Do(func() { close(started) })
		<-release
		w.WriteHeader(http.StatusCreated)
	}, false)
	done := make(chan *httptest.ResponseRecorder)
	go func() { done <- do(h, "key-00000005", "u1", `{}`) }()
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("first request did not start")
	}
	dup := do(h, "key-00000005", "u1", `{}`)
	close(release)
	first := <-done
	if dup.Code != http.StatusConflict || !strings.Contains(dup.Body.String(), "request_in_progress") || dup.Header().Get("Retry-After") == "" {
		t.Fatalf("duplicate: %d %s", dup.Code, dup.Body.String())
	}
	if first.Code != http.StatusCreated {
		t.Fatalf("first=%d", first.Code)
	}
}

func TestRequiredHeaderAndValidation(t *testing.T) {
	h, _ := setup(t, func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(201) }, true)
	if rec := do(h, "", "u1", `{}`); rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("missing key: %d", rec.Code)
	}
	if rec := do(h, "short", "u1", `{}`); rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("short key: %d", rec.Code)
	}
}

func TestPurgeExpired(t *testing.T) {
	h, d := setup(t, func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(201) }, false)
	do(h, "key-00000006", "u1", `{}`)
	d.Pool("superuser").Exec(context.Background(), `UPDATE platform.idempotency_keys SET expires_at = now() - interval '1 minute'`) //nolint:errcheck
	tx, err := d.Pool("app_worker").Begin(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	n, err := idempotency.Purge(context.Background(), tx, 100)
	if err != nil || n != 1 {
		t.Fatalf("purged %d err=%v", n, err)
	}
	tx.Commit(context.Background()) //nolint:errcheck
}
