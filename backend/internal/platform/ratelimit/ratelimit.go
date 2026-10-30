// Package ratelimit implements fixed-window rate limits in Valkey/Redis (ADR-005, doc 07 §4.1): OTP abuse
// prevention, authentication attempts and content creation. Counters live only in Valkey; nothing here is a
// system of record.
package ratelimit

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/NADER-CY/vira/backend/internal/platform/apperr"
)

// Rule is one limit: at most Limit events per Window for a key.
type Rule struct {
	Name   string
	Limit  int
	Window time.Duration
}

// Limiter checks and counts events.
type Limiter struct {
	Client *redis.Client
	Prefix string // key prefix, e.g. "vira:rl:"
	Now    func() time.Time
}

// New connects to Valkey at addr.
func New(addr string) *Limiter {
	return &Limiter{Client: redis.NewClient(&redis.Options{Addr: addr, DialTimeout: 2 * time.Second,
		ReadTimeout: time.Second, WriteTimeout: time.Second}), Prefix: "vira:rl:"}
}

// ErrUnavailable is returned when the limiter store cannot be reached. Callers decide whether to fail
// closed (authentication) or open.
var ErrUnavailable = errors.New("ratelimit: store unavailable")

var incr = redis.NewScript(`
local n = redis.call('INCR', KEYS[1])
if n == 1 then redis.call('PEXPIRE', KEYS[1], ARGV[1]) end
return {n, redis.call('PTTL', KEYS[1])}`)

// Result reports the outcome of Allow.
type Result struct {
	Allowed    bool
	Count      int
	RetryAfter time.Duration
}

// Allow counts one event for key under rule and reports whether it is within the limit.
func (l *Limiter) Allow(ctx context.Context, r Rule, key string) (Result, error) {
	now := time.Now
	if l.Now != nil {
		now = l.Now
	}
	window := now().UnixMilli() / r.Window.Milliseconds()
	k := fmt.Sprintf("%s%s:%s:%d", l.Prefix, r.Name, key, window)
	res, err := incr.Run(ctx, l.Client, []string{k}, r.Window.Milliseconds()).Int64Slice()
	if err != nil || len(res) != 2 {
		return Result{}, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	ttl := time.Duration(res[1]) * time.Millisecond
	if ttl < 0 {
		ttl = r.Window
	}
	return Result{Allowed: res[0] <= int64(r.Limit), Count: int(res[0]), RetryAfter: ttl}, nil
}

// Check applies several rules (all are counted) and returns a 429 rate_limited error with retry_after when any
// is exceeded. When the store is unavailable it fails closed with 503 if failClosed, otherwise it allows.
func (l *Limiter) Check(ctx context.Context, failClosed bool, checks ...Keyed) error {
	var worst time.Duration
	limited := false
	for _, c := range checks {
		if c.Key == "" {
			continue
		}
		res, err := l.Allow(ctx, c.Rule, c.Key)
		if err != nil {
			if failClosed {
				return apperr.Wrap(apperr.Unavailable, err)
			}
			continue
		}
		if !res.Allowed {
			limited = true
			if res.RetryAfter > worst {
				worst = res.RetryAfter
			}
		}
	}
	if limited {
		e := apperr.New(apperr.RateLimited, "")
		e.Extra = map[string]any{"retry_after": int(worst.Seconds()) + 1}
		return e
	}
	return nil
}

// Keyed pairs a rule with the key it applies to.
type Keyed struct {
	Rule Rule
	Key  string
}

// Ping checks connectivity (readiness).
func (l *Limiter) Ping(ctx context.Context) error { return l.Client.Ping(ctx).Err() }
