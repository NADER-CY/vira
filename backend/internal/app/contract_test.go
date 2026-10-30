// Package app_test checks the Go routers against the OpenAPI contracts in /contracts (doc 07).
package app_test

import (
	"crypto/ecdsa"
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/NADER-CY/vira/backend/internal/app/adminapi"
	"github.com/NADER-CY/vira/backend/internal/app/publicapi"
	"github.com/NADER-CY/vira/backend/internal/app/realtime"
	"github.com/NADER-CY/vira/backend/internal/modules/identity"
	"github.com/NADER-CY/vira/backend/internal/modules/media"
	"github.com/NADER-CY/vira/backend/internal/modules/moderation"
	"github.com/NADER-CY/vira/backend/internal/modules/social"
	"github.com/NADER-CY/vira/backend/internal/platform/apperr"
	"github.com/NADER-CY/vira/backend/internal/platform/auth"
	"github.com/NADER-CY/vira/backend/internal/platform/health"
	"github.com/NADER-CY/vira/backend/internal/platform/httpx"
	"github.com/NADER-CY/vira/backend/internal/platform/outbox"
)

func contract(t *testing.T, name string) map[string]any {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "..", "..", "contracts", name))
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(b, &doc); err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	return doc
}

func specRoutes(doc map[string]any) []string {
	var out []string
	for path, ops := range doc["paths"].(map[string]any) {
		for method := range ops.(map[string]any) {
			out = append(out, strings.ToUpper(method)+" "+path)
		}
	}
	sort.Strings(out)
	return out
}

func codeRoutes(rs []httpx.Route) []string {
	var out []string
	for _, r := range rs {
		out = append(out, r.Method+" "+r.Path)
	}
	sort.Strings(out)
	return out
}

func diff(t *testing.T, name string, code, spec []string) {
	t.Helper()
	in := func(s []string, x string) bool {
		for _, y := range s {
			if y == x {
				return true
			}
		}
		return false
	}
	for _, r := range code {
		if !in(spec, r) {
			t.Errorf("%s: route %q is served but missing from the contract", name, r)
		}
	}
	for _, r := range spec {
		if !in(code, r) {
			t.Errorf("%s: contract route %q is not served", name, r)
		}
	}
}

func verifier() *auth.Verifier {
	k, _ := auth.GenerateKey()
	return &auth.Verifier{Keys: map[string]*ecdsa.PublicKey{"k": &k.PublicKey}, Issuer: "vira", Audience: "vira-api"}
}

func TestPublicRoutesMatchContract(t *testing.T) {
	r := publicapi.New(publicapi.Deps{Verifier: verifier(), Health: &health.Handler{}, Identity: &identity.Service{},
		Social: &social.Service{}, Media: &media.Service{}, Moderation: &moderation.Service{}})
	diff(t, "public.v1", codeRoutes(r.Routes()), specRoutes(contract(t, "openapi/public.v1.json")))
}

func TestAdminRoutesMatchContract(t *testing.T) {
	r := adminapi.New(adminapi.Deps{Health: &health.Handler{}})
	diff(t, "admin.v1", codeRoutes(r.Routes()), specRoutes(contract(t, "openapi/admin.v1.json")))
}

func TestRealtimeRoutes(t *testing.T) {
	r := realtime.New(realtime.Deps{Verifier: verifier(), Health: &health.Handler{}, Log: slog.New(slog.NewTextHandler(io.Discard, nil))})
	got := codeRoutes(r.Routes())
	want := []string{"GET /health/live", "GET /health/ready", "GET /v1/ws"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("realtime routes %v, want %v", got, want)
	}
}

func TestErrorCodeEnumMatchesBackend(t *testing.T) {
	var backend []string
	for _, c := range apperr.Codes() {
		backend = append(backend, string(c))
	}
	sort.Strings(backend)
	for _, name := range []string{"openapi/public.v1.json", "openapi/admin.v1.json"} {
		schemas := contract(t, name)["components"].(map[string]any)["schemas"].(map[string]any)
		enum := schemas["Problem"].(map[string]any)["properties"].(map[string]any)["code"].(map[string]any)["enum"].([]any)
		var spec []string
		for _, e := range enum {
			spec = append(spec, e.(string))
		}
		sort.Strings(spec)
		if strings.Join(spec, ",") != strings.Join(backend, ",") {
			t.Errorf("%s: Problem.code enum %v differs from backend codes %v", name, spec, backend)
		}
	}
}

func TestEventEnvelopeMatchesSchema(t *testing.T) {
	schema := contract(t, "events/envelope.v1.schema.json")
	env := outbox.Envelope{SpecVersion: "1.0", ID: "0192f1c6-7b7a-7c1e-9d2f-5a1b3c4d5e6f", Source: "vira/orders",
		Type: "vira.orders.order_delivered.v1", Subject: "seller_order/1", Time: time.Now().UTC(),
		DataContentType: "application/json", AggregateID: "1", AggregateVersion: 7, Data: json.RawMessage(`{}`)}
	b, _ := json.Marshal(env)
	var m map[string]any
	_ = json.Unmarshal(b, &m)
	props := schema["properties"].(map[string]any)
	for _, req := range schema["required"].([]any) {
		if _, ok := m[req.(string)]; !ok {
			t.Errorf("envelope is missing required field %s", req)
		}
	}
	for k := range m {
		if _, ok := props[k]; !ok {
			t.Errorf("envelope field %s is not in the schema (additionalProperties: false)", k)
		}
	}
}
