package httpx

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/NADER-CY/vira/backend/internal/platform/apperr"
)

func TestProblemJSONShape(t *testing.T) {
	e := apperr.New(apperr.FeatureLocked, "requires FG-1b")
	e.Extra = map[string]any{"gate": "FG-1b", "code": "must-not-override"}
	rec := httptest.NewRecorder()
	WriteError(rec, httptest.NewRequest("GET", "/", nil), e)
	var m map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &m); err != nil {
		t.Fatal(err)
	}
	if rec.Code != 409 || m["type"] != "urn:vira:error:feature_locked" || m["code"] != "feature_locked" ||
		m["gate"] != "FG-1b" || m["status"].(float64) != 409 || m["title"] == "" {
		t.Fatalf("problem: %d %v", rec.Code, m)
	}
}

func TestInternalErrorsDoNotLeak(t *testing.T) {
	rec := httptest.NewRecorder()
	WriteError(rec, httptest.NewRequest("GET", "/", nil), io.ErrUnexpectedEOF)
	if rec.Code != 500 || strings.Contains(rec.Body.String(), "unexpected EOF") {
		t.Fatalf("internal error leaked: %s", rec.Body.String())
	}
}

func TestDecodeJSONIsStrict(t *testing.T) {
	type in struct {
		A string `json:"a"`
	}
	cases := map[string]string{"unknown field": `{"a":"x","b":1}`, "trailing": `{"a":"x"}{}`, "not json": `x`}
	for name, body := range cases {
		r := httptest.NewRequest("POST", "/", strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		var v in
		if err := DecodeJSON(r, &v); err == nil {
			t.Errorf("%s accepted", name)
		}
	}
	r := httptest.NewRequest("POST", "/", strings.NewReader(`{"a":"x"}`))
	r.Header.Set("Content-Type", "text/plain")
	var v in
	if err := DecodeJSON(r, &v); err == nil {
		t.Error("wrong content type accepted")
	}
}

func TestRecoverAndRequestContext(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	h := Chain(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic("boom") }), RequestContext(log), Recover)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("traceparent", "garbage")
	h.ServeHTTP(rec, req)
	if rec.Code != 500 || rec.Header().Get("X-Request-Id") == "" || !strings.HasPrefix(rec.Header().Get("traceparent"), "00-") {
		t.Fatalf("got %d headers=%v", rec.Code, rec.Header())
	}
}

func TestCORSAllowList(t *testing.T) {
	h := CORS([]string{"https://admin.vira.test"})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) }))
	req := httptest.NewRequest("OPTIONS", "/", nil)
	req.Header.Set("Origin", "https://admin.vira.test")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Header().Get("Access-Control-Allow-Origin") != "https://admin.vira.test" || rec.Header().Get("Access-Control-Allow-Credentials") != "true" {
		t.Fatalf("allowed origin: %v", rec.Header())
	}
	req.Header.Set("Origin", "https://evil.test")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatal("unlisted origin allowed")
	}
}

func TestRouterRecordsRoutesAndRendersProblems(t *testing.T) {
	r := NewRouter()
	r.HandleFunc("GET", "/v1/a/{id}", func(w http.ResponseWriter, req *http.Request) { w.Write([]byte(req.PathValue("id"))) }) //nolint:errcheck
	r.HandleFunc("POST", "/v1/a/{id}", func(w http.ResponseWriter, req *http.Request) {})
	if len(r.Routes()) != 2 || r.Routes()[0].Method != "GET" {
		t.Fatalf("routes: %v", r.Routes())
	}
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest("GET", "/v1/a/42", nil))
	if rec.Body.String() != "42" {
		t.Fatalf("path value: %q", rec.Body.String())
	}
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest("DELETE", "/v1/a/42", nil))
	if rec.Code != 405 || !strings.Contains(rec.Header().Get("Allow"), "GET") || !strings.Contains(rec.Body.String(), "method_not_allowed") {
		t.Fatalf("405: %d %v %s", rec.Code, rec.Header(), rec.Body.String())
	}
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest("GET", "/v2/x", nil))
	if rec.Code != 404 || rec.Header().Get("Content-Type") != "application/problem+json" {
		t.Fatalf("404: %d %v", rec.Code, rec.Header())
	}
}

func TestEveryErrorCodeHasPersianTitle(t *testing.T) {
	for _, c := range apperr.Codes() {
		title := apperr.Title(c)
		if title == "" || !strings.ContainsAny(title, "اآبپتثجچحخدذرزژسشصضطظعغفقکگلمنوهی") {
			t.Errorf("code %s has no Persian title", c)
		}
	}
}
