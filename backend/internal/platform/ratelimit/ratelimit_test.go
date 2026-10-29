package ratelimit

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	"github.com/NADER-CY/vira/backend/internal/platform/apperr"
)

func newLimiter(t *testing.T) (*Limiter, *miniredis.Miniredis) {
	mr := miniredis.RunT(t)
	return &Limiter{Client: redis.NewClient(&redis.Options{Addr: mr.Addr()}), Prefix: "t:"}, mr
}

func TestFixedWindow(t *testing.T) {
	l, mr := newLimiter(t)
	base := time.Date(2026, 10, 29, 10, 0, 0, 0, time.UTC)
	l.Now = func() time.Time { return base }
	r := Rule{Name: "otp_phone", Limit: 3, Window: 10 * time.Minute}
	for i := 1; i <= 3; i++ {
		if res, err := l.Allow(context.Background(), r, "p1"); err != nil || !res.Allowed {
			t.Fatalf("event %d: %+v %v", i, res, err)
		}
	}
	res, _ := l.Allow(context.Background(), r, "p1")
	if res.Allowed || res.RetryAfter <= 0 {
		t.Fatalf("4th event allowed: %+v", res)
	}
	if res, _ := l.Allow(context.Background(), r, "p2"); !res.Allowed {
		t.Fatal("other key affected")
	}
	// Next window.
	mr.FastForward(10 * time.Minute)
	l.Now = func() time.Time { return base.Add(10 * time.Minute) }
	if res, _ := l.Allow(context.Background(), r, "p1"); !res.Allowed {
		t.Fatal("new window still limited")
	}
}

func TestCheckReturnsRateLimitedWithRetryAfter(t *testing.T) {
	l, _ := newLimiter(t)
	r := Rule{Name: "x", Limit: 1, Window: time.Minute}
	if err := l.Check(context.Background(), true, Keyed{r, "a"}); err != nil {
		t.Fatal(err)
	}
	err := l.Check(context.Background(), true, Keyed{r, "a"}, Keyed{Rule{Name: "y", Limit: 9, Window: time.Hour}, "b"})
	e := apperr.As(err)
	if e == nil || e.Code != apperr.RateLimited || e.Extra["retry_after"].(int) < 1 {
		t.Fatalf("got %v", err)
	}
}

func TestStoreDownFailsClosedOrOpen(t *testing.T) {
	l, mr := newLimiter(t)
	mr.Close()
	r := Rule{Name: "x", Limit: 1, Window: time.Minute}
	if err := l.Check(context.Background(), true, Keyed{r, "a"}); apperr.As(err) == nil || apperr.As(err).Code != apperr.Unavailable {
		t.Fatalf("fail closed: %v", err)
	}
	if err := l.Check(context.Background(), false, Keyed{r, "a"}); err != nil {
		t.Fatalf("fail open: %v", err)
	}
	if _, err := l.Allow(context.Background(), r, "a"); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("Allow: %v", err)
	}
}
