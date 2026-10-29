package secure

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestPasswordHashRoundTrip(t *testing.T) {
	h, err := HashPassword("correct horse battery")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(h, "$argon2id$v=19$") {
		t.Fatalf("unexpected encoding %s", h)
	}
	if !VerifyPassword(h, "correct horse battery") {
		t.Fatal("valid password rejected")
	}
	if VerifyPassword(h, "correct horse batterY") {
		t.Fatal("wrong password accepted")
	}
	if VerifyPassword("$bcrypt$x", "x") {
		t.Fatal("foreign hash accepted")
	}
	if _, err := HashPassword("short"); err == nil {
		t.Fatal("short password accepted")
	}
	h2, _ := HashPassword("correct horse battery")
	if h == h2 {
		t.Fatal("salt not random")
	}
}

// RFC 6238 Appendix B test vectors (SHA1, secret "12345678901234567890"), truncated to 6 digits.
func TestTOTPRFC6238Vectors(t *testing.T) {
	secret := "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ" // base32("12345678901234567890")
	vectors := map[int64]string{59: "287082", 1111111109: "081804", 1111111111: "050471", 1234567890: "005924", 2000000000: "279037"}
	for ts, want := range vectors {
		got, err := TOTPCode(secret, TOTPStep(time.Unix(ts, 0)))
		if err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Errorf("t=%d got %s want %s", ts, got, want)
		}
	}
}

func TestVerifyTOTPWindowAndReplay(t *testing.T) {
	secret := NewTOTPSecret()
	now := time.Unix(1_800_000_000, 0)
	step := TOTPStep(now)
	code, _ := TOTPCode(secret, step)
	got, ok := VerifyTOTP(secret, code, now, 0)
	if !ok || got != step {
		t.Fatal("current code rejected")
	}
	if _, ok := VerifyTOTP(secret, code, now, step); ok {
		t.Fatal("replayed code accepted")
	}
	prev, _ := TOTPCode(secret, step-1)
	if _, ok := VerifyTOTP(secret, prev, now, 0); !ok {
		t.Fatal("previous-step code rejected (clock skew tolerance)")
	}
	old, _ := TOTPCode(secret, step-3)
	if _, ok := VerifyTOTP(secret, old, now, 0); ok {
		t.Fatal("stale code accepted")
	}
	if _, ok := VerifyTOTP(secret, "12345", now, 0); ok {
		t.Fatal("5-digit code accepted")
	}
	if !strings.HasPrefix(TOTPURI("VIRA Admin", "a@b.c", secret), "otpauth://totp/VIRA%20Admin:a@b.c?") {
		t.Fatal(TOTPURI("VIRA Admin", "a@b.c", secret))
	}
}

func TestSealOpen(t *testing.T) {
	key := bytes.Repeat([]byte{7}, 32)
	sealed, err := Seal(key, []byte("secret"), []byte("admin:1"))
	if err != nil {
		t.Fatal(err)
	}
	plain, err := Open(key, sealed, []byte("admin:1"))
	if err != nil || string(plain) != "secret" {
		t.Fatalf("open: %q %v", plain, err)
	}
	if _, err := Open(key, sealed, []byte("admin:2")); err == nil {
		t.Fatal("ciphertext moved to another record must not decrypt")
	}
	if _, err := Open(bytes.Repeat([]byte{8}, 32), sealed, []byte("admin:1")); err == nil {
		t.Fatal("wrong key decrypted")
	}
	if _, err := Seal([]byte("short"), []byte("x"), nil); err == nil {
		t.Fatal("short key accepted")
	}
}

func TestTokens(t *testing.T) {
	tok, h := NewToken()
	if len(tok) < 40 || !bytes.Equal(h, HashToken(tok)) {
		t.Fatal("token/hash mismatch")
	}
	tok2, _ := NewToken()
	if tok == tok2 {
		t.Fatal("tokens repeat")
	}
}
