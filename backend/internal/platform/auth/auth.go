// Package auth is the public-realm authentication framework (doc 07 §1, doc 08):
// short-lived ES256 access JWTs with key ids, a request Principal, and middleware.
//
// Token issuance from phone OTP arrives in Phase 2 (identity module); Phase 1 provides signing,
// verification, key loading and the middleware every protected endpoint uses.
package auth

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/NADER-CY/vira/backend/internal/platform/apperr"
	"github.com/NADER-CY/vira/backend/internal/platform/httpx"
	"github.com/NADER-CY/vira/backend/internal/platform/ids"
)

// Principal is the authenticated caller.
type Principal struct {
	Subject string    `json:"sub"`  // user id
	Kind    string    `json:"kind"` // user (admin realm is separate, package adminauth)
	Scopes  []string  `json:"scopes,omitempty"`
	TokenID string    `json:"-"`
	Expires time.Time `json:"-"`
}

// HasScope reports whether the principal carries scope s.
func (p Principal) HasScope(s string) bool {
	for _, x := range p.Scopes {
		if x == s {
			return true
		}
	}
	return false
}

type ctxKey struct{}

// WithPrincipal stores the principal in ctx.
func WithPrincipal(ctx context.Context, p Principal) context.Context {
	return context.WithValue(ctx, ctxKey{}, p)
}

// FromContext returns the principal, if authenticated.
func FromContext(ctx context.Context) (Principal, bool) {
	p, ok := ctx.Value(ctxKey{}).(Principal)
	return p, ok
}

// Claims are the JWT claims VIRA issues.
type Claims struct {
	Issuer    string   `json:"iss"`
	Subject   string   `json:"sub"`
	Audience  string   `json:"aud"`
	IssuedAt  int64    `json:"iat"`
	NotBefore int64    `json:"nbf"`
	Expires   int64    `json:"exp"`
	ID        string   `json:"jti"`
	Kind      string   `json:"kind"`
	Scopes    []string `json:"scp,omitempty"`
}

// Signer issues ES256 tokens.
type Signer struct {
	Key      *ecdsa.PrivateKey
	KeyID    string
	Issuer   string
	Audience string
	TTL      time.Duration
	Now      func() time.Time
}

// Verifier validates ES256 tokens against one or more public keys (key rotation by kid).
type Verifier struct {
	Keys     map[string]*ecdsa.PublicKey
	Issuer   string
	Audience string
	Leeway   time.Duration
	Now      func() time.Time
}

var b64 = base64.RawURLEncoding

// KeyID derives a stable key id from the public key.
func KeyID(pub *ecdsa.PublicKey) string {
	der, _ := x509.MarshalPKIXPublicKey(pub)
	s := sha256.Sum256(der)
	return b64.EncodeToString(s[:8])
}

// Issue creates an access token for subject.
func (s *Signer) Issue(subject, kind string, scopes []string) (string, Claims, error) {
	now := time.Now
	if s.Now != nil {
		now = s.Now
	}
	t := now().UTC()
	c := Claims{Issuer: s.Issuer, Subject: subject, Audience: s.Audience, IssuedAt: t.Unix(), NotBefore: t.Unix(),
		Expires: t.Add(s.TTL).Unix(), ID: ids.New(), Kind: kind, Scopes: scopes}
	hdr, _ := json.Marshal(map[string]string{"alg": "ES256", "typ": "JWT", "kid": s.KeyID})
	body, err := json.Marshal(c)
	if err != nil {
		return "", Claims{}, err
	}
	signing := b64.EncodeToString(hdr) + "." + b64.EncodeToString(body)
	digest := sha256.Sum256([]byte(signing))
	r, sv, err := ecdsa.Sign(rand.Reader, s.Key, digest[:])
	if err != nil {
		return "", Claims{}, err
	}
	sig := make([]byte, 64) // JWS ES256: R || S, each 32 bytes big-endian
	r.FillBytes(sig[:32])
	sv.FillBytes(sig[32:])
	return signing + "." + b64.EncodeToString(sig), c, nil
}

// ErrInvalidToken is returned for any invalid token (reason intentionally not exposed to clients).
var ErrInvalidToken = errors.New("auth: invalid token")

// Verify parses and validates a token.
func (v *Verifier) Verify(token string) (Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return Claims{}, ErrInvalidToken
	}
	hb, err := b64.DecodeString(parts[0])
	if err != nil {
		return Claims{}, ErrInvalidToken
	}
	var hdr struct {
		Alg string `json:"alg"`
		Kid string `json:"kid"`
		Typ string `json:"typ"`
	}
	if err := json.Unmarshal(hb, &hdr); err != nil || hdr.Alg != "ES256" { // never accept "none" or HMAC
		return Claims{}, ErrInvalidToken
	}
	pub, ok := v.Keys[hdr.Kid]
	if !ok {
		return Claims{}, ErrInvalidToken
	}
	sig, err := b64.DecodeString(parts[2])
	if err != nil || len(sig) != 64 {
		return Claims{}, ErrInvalidToken
	}
	digest := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	r, s := new(big.Int).SetBytes(sig[:32]), new(big.Int).SetBytes(sig[32:])
	if !ecdsa.Verify(pub, digest[:], r, s) {
		return Claims{}, ErrInvalidToken
	}
	cb, err := b64.DecodeString(parts[1])
	if err != nil {
		return Claims{}, ErrInvalidToken
	}
	var c Claims
	if err := json.Unmarshal(cb, &c); err != nil {
		return Claims{}, ErrInvalidToken
	}
	now := time.Now
	if v.Now != nil {
		now = v.Now
	}
	t := now().Unix()
	leeway := int64(v.Leeway.Seconds())
	switch {
	case c.Issuer != v.Issuer, c.Audience != v.Audience, c.Subject == "":
		return Claims{}, ErrInvalidToken
	case t > c.Expires+leeway, t+leeway < c.NotBefore:
		return Claims{}, ErrInvalidToken
	}
	return c, nil
}

// Middleware authenticates bearer tokens. With required=false, anonymous requests pass through.
func (v *Verifier) Middleware(required bool) httpx.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := r.Header.Get("Authorization")
			if h == "" {
				if required {
					w.Header().Set("WWW-Authenticate", `Bearer realm="vira"`)
					httpx.WriteError(w, r, apperr.New(apperr.Unauthenticated, ""))
					return
				}
				next.ServeHTTP(w, r)
				return
			}
			tok, ok := strings.CutPrefix(h, "Bearer ")
			if !ok {
				httpx.WriteError(w, r, apperr.New(apperr.Unauthenticated, ""))
				return
			}
			c, err := v.Verify(strings.TrimSpace(tok))
			if err != nil {
				w.Header().Set("WWW-Authenticate", `Bearer realm="vira", error="invalid_token"`)
				httpx.WriteError(w, r, apperr.New(apperr.Unauthenticated, ""))
				return
			}
			p := Principal{Subject: c.Subject, Kind: c.Kind, Scopes: c.Scopes, TokenID: c.ID, Expires: time.Unix(c.Expires, 0)}
			next.ServeHTTP(w, r.WithContext(WithPrincipal(r.Context(), p)))
		})
	}
}

// RequireScope rejects authenticated principals without scope (403); unauthenticated → 401.
func RequireScope(scope string) httpx.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			p, ok := FromContext(r.Context())
			if !ok {
				httpx.WriteError(w, r, apperr.New(apperr.Unauthenticated, ""))
				return
			}
			if !p.HasScope(scope) {
				httpx.WriteError(w, r, apperr.New(apperr.Forbidden, ""))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// GenerateKey creates a new P-256 key (development and tests).
func GenerateKey() (*ecdsa.PrivateKey, error) { return ecdsa.GenerateKey(elliptic.P256(), rand.Reader) }

// LoadPrivateKey reads a PEM (PKCS#8 or SEC1) EC private key.
func LoadPrivateKey(path string) (*ecdsa.PrivateKey, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	blk, _ := pem.Decode(b)
	if blk == nil {
		return nil, fmt.Errorf("auth: %s: no PEM block", path)
	}
	if k, err := x509.ParsePKCS8PrivateKey(blk.Bytes); err == nil {
		if ek, ok := k.(*ecdsa.PrivateKey); ok && ek.Curve == elliptic.P256() {
			return ek, nil
		}
		return nil, errors.New("auth: key is not a P-256 EC key")
	}
	return x509.ParseECPrivateKey(blk.Bytes)
}

// LoadPublicKey reads a PEM PKIX EC public key.
func LoadPublicKey(path string) (*ecdsa.PublicKey, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	blk, _ := pem.Decode(b)
	if blk == nil {
		return nil, fmt.Errorf("auth: %s: no PEM block", path)
	}
	k, err := x509.ParsePKIXPublicKey(blk.Bytes)
	if err != nil {
		return nil, err
	}
	ek, ok := k.(*ecdsa.PublicKey)
	if !ok || ek.Curve != elliptic.P256() {
		return nil, errors.New("auth: key is not a P-256 EC public key")
	}
	return ek, nil
}
