package auth

import (
	"crypto/ecdsa"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func pair(t *testing.T) (*Signer, *Verifier) {
	t.Helper()
	k, err := GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	kid := KeyID(&k.PublicKey)
	now := time.Unix(1_800_000_000, 0)
	s := &Signer{Key: k, KeyID: kid, Issuer: "vira", Audience: "vira-api", TTL: 10 * time.Minute, Now: func() time.Time { return now }}
	v := &Verifier{Keys: map[string]*ecdsa.PublicKey{kid: &k.PublicKey}, Issuer: "vira", Audience: "vira-api", Now: func() time.Time { return now }}
	return s, v
}

func TestIssueVerify(t *testing.T) {
	s, v := pair(t)
	tok, _, err := s.Issue("user-1", "user", []string{"profile"})
	if err != nil {
		t.Fatal(err)
	}
	c, err := v.Verify(tok)
	if err != nil || c.Subject != "user-1" || c.Kind != "user" || len(c.Scopes) != 1 {
		t.Fatalf("claims=%+v err=%v", c, err)
	}
}

func TestRejections(t *testing.T) {
	s, v := pair(t)
	tok, _, _ := s.Issue("user-1", "user", nil)
	parts := strings.Split(tok, ".")

	// expired
	late := *v
	late.Now = func() time.Time { return time.Unix(1_800_000_000, 0).Add(11 * time.Minute) }
	if _, err := late.Verify(tok); err == nil {
		t.Error("expired token accepted")
	}
	// wrong audience
	aud := *v
	aud.Audience = "other"
	if _, err := aud.Verify(tok); err == nil {
		t.Error("wrong audience accepted")
	}
	// tampered payload
	body, _ := base64.RawURLEncoding.DecodeString(parts[1])
	forged := base64.RawURLEncoding.EncodeToString([]byte(strings.Replace(string(body), "user-1", "user-2", 1)))
	if _, err := v.Verify(parts[0] + "." + forged + "." + parts[2]); err == nil {
		t.Error("tampered token accepted")
	}
	// alg=none
	none := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none","typ":"JWT"}`))
	if _, err := v.Verify(none + "." + parts[1] + "."); err == nil {
		t.Error("alg=none accepted")
	}
	// unknown key id (key from another signer)
	other, _ := pair(t)
	tok2, _, _ := other.Issue("user-1", "user", nil)
	if _, err := v.Verify(tok2); err == nil {
		t.Error("token from unknown key accepted")
	}
}

func TestMiddleware(t *testing.T) {
	s, v := pair(t)
	tok, _, _ := s.Issue("user-9", "user", []string{"seller"})
	var seen Principal
	h := v.Middleware(true)(RequireScope("seller")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen, _ = FromContext(r.Context())
		w.WriteHeader(204)
	})))
	cases := []struct {
		header string
		want   int
	}{
		{"", 401},
		{"Basic abc", 401},
		{"Bearer garbage", 401},
		{"Bearer " + tok, 204},
	}
	for _, c := range cases {
		req := httptest.NewRequest("GET", "/v1/me", nil)
		if c.header != "" {
			req.Header.Set("Authorization", c.header)
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != c.want {
			t.Errorf("%q: got %d want %d", c.header, rec.Code, c.want)
		}
		if rec.Code == 401 && rec.Header().Get("Content-Type") != "application/problem+json" {
			t.Errorf("401 must be problem+json")
		}
	}
	if seen.Subject != "user-9" {
		t.Fatalf("principal not propagated: %+v", seen)
	}
	noScope, _, _ := s.Issue("user-9", "user", nil)
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+noScope)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != 403 {
		t.Fatalf("missing scope: %d", rec.Code)
	}
}
