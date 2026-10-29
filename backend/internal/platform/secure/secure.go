// Package secure contains small, reviewed cryptographic helpers: Argon2id password hashing, TOTP (RFC 6238),
// AES-256-GCM sealing of secrets at rest and opaque session tokens.
package secure

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1" //nolint:gosec // RFC 6238 TOTP uses HMAC-SHA1 for authenticator-app compatibility
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base32"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"golang.org/x/crypto/argon2"
)

// Argon2id parameters (OWASP 2024+: m=19 MiB, t=2, p=1 minimum; we use 64 MiB, t=3).
const (
	argonTime    = 3
	argonMemory  = 64 * 1024
	argonThreads = 2
	argonKeyLen  = 32
	saltLen      = 16
)

// HashPassword returns an encoded Argon2id hash ($argon2id$v=19$m=...,t=...,p=...$salt$hash).
func HashPassword(password string) (string, error) {
	if len(password) < 12 {
		return "", errors.New("secure: password must be at least 12 characters")
	}
	salt := make([]byte, saltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	key := argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonThreads, argonKeyLen)
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s", argon2.Version, argonMemory, argonTime, argonThreads,
		base64.RawStdEncoding.EncodeToString(salt), base64.RawStdEncoding.EncodeToString(key)), nil
}

// VerifyPassword checks password against an encoded hash in constant time.
func VerifyPassword(encoded, password string) bool {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return false
	}
	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil || version != argon2.Version {
		return false
	}
	var m uint32
	var t uint32
	var p uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &m, &t, &p); err != nil {
		return false
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false
	}
	want, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false
	}
	got := argon2.IDKey([]byte(password), salt, t, m, p, uint32(len(want)))
	return subtle.ConstantTimeCompare(got, want) == 1
}

// DummyVerify burns comparable time when the account does not exist (prevents user enumeration by timing).
func DummyVerify(password string) {
	salt := make([]byte, saltLen)
	argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonThreads, argonKeyLen)
}

// ---------------------------------------------------------------------------------------------
// TOTP (RFC 6238): 30-second steps, 6 digits, HMAC-SHA1, ±1 step tolerance.

const totpPeriod = 30

// NewTOTPSecret returns a random 20-byte secret, base32-encoded without padding.
func NewTOTPSecret() string {
	b := make([]byte, 20)
	_, _ = rand.Read(b)
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(b)
}

// TOTPURI returns the otpauth:// URI for authenticator apps.
func TOTPURI(issuer, account, secret string) string {
	label := url.PathEscape(issuer + ":" + account)
	v := url.Values{"secret": {secret}, "issuer": {issuer}, "algorithm": {"SHA1"}, "digits": {"6"}, "period": {"30"}}
	return "otpauth://totp/" + label + "?" + v.Encode()
}

// TOTPCode computes the code for a time step.
func TOTPCode(secret string, step int64) (string, error) {
	key, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.ToUpper(strings.TrimSpace(secret)))
	if err != nil {
		return "", fmt.Errorf("secure: bad TOTP secret: %w", err)
	}
	var msg [8]byte
	binary.BigEndian.PutUint64(msg[:], uint64(step))
	mac := hmac.New(sha1.New, key)
	mac.Write(msg[:])
	sum := mac.Sum(nil)
	off := sum[len(sum)-1] & 0x0f
	bin := (uint32(sum[off])&0x7f)<<24 | uint32(sum[off+1])<<16 | uint32(sum[off+2])<<8 | uint32(sum[off+3])
	return fmt.Sprintf("%06d", bin%1_000_000), nil
}

// TOTPStep returns the time step for t.
func TOTPStep(t time.Time) int64 { return t.Unix() / totpPeriod }

// VerifyTOTP checks code against the current step ±1 and returns the matched step. Steps at or below
// lastUsed are rejected so that a code cannot be replayed.
func VerifyTOTP(secret, code string, now time.Time, lastUsed int64) (int64, bool) {
	code = strings.TrimSpace(code)
	if len(code) != 6 {
		return 0, false
	}
	cur := TOTPStep(now)
	for _, s := range []int64{cur - 1, cur, cur + 1} {
		if s <= lastUsed {
			continue
		}
		want, err := TOTPCode(secret, s)
		if err != nil {
			return 0, false
		}
		if subtle.ConstantTimeCompare([]byte(want), []byte(code)) == 1 {
			return s, true
		}
	}
	return 0, false
}

// ---------------------------------------------------------------------------------------------
// AES-256-GCM sealing for secrets stored in the database (key held outside the database).

// Seal encrypts plaintext with a 32-byte key; output = nonce || ciphertext.
func Seal(key, plaintext, additional []byte) ([]byte, error) {
	gcm, err := newGCM(key)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	return gcm.Seal(nonce, nonce, plaintext, additional), nil
}

// Open decrypts the output of Seal.
func Open(key, sealed, additional []byte) ([]byte, error) {
	gcm, err := newGCM(key)
	if err != nil {
		return nil, err
	}
	if len(sealed) < gcm.NonceSize() {
		return nil, errors.New("secure: sealed data too short")
	}
	return gcm.Open(nil, sealed[:gcm.NonceSize()], sealed[gcm.NonceSize():], additional)
}

func newGCM(key []byte) (cipher.AEAD, error) {
	if len(key) != 32 {
		return nil, errors.New("secure: key must be 32 bytes")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

// ---------------------------------------------------------------------------------------------
// Opaque tokens.

// NewToken returns a 256-bit random token (URL-safe) and its SHA-256 for storage.
func NewToken() (token string, hash []byte) {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	token = base64.RawURLEncoding.EncodeToString(b)
	return token, HashToken(token)
}

// HashToken returns the SHA-256 of a token (only hashes are stored).
func HashToken(token string) []byte {
	s := sha256.Sum256([]byte(token))
	return s[:]
}
