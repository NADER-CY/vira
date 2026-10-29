// Package health implements liveness and readiness endpoints and dependency checks.
//
//	GET /health/live   — process is up (no dependency checks)
//	GET /health/ready  — all required dependencies answer within the timeout
package health

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/NADER-CY/vira/backend/internal/platform/httpx"
)

// Check is one named dependency check.
type Check struct {
	Name string
	Run  func(ctx context.Context) error
}

// Handler serves liveness and readiness.
type Handler struct {
	Service string
	Version string
	Checks  []Check
	Timeout time.Duration
}

// Register mounts the health endpoints on r.
func (h *Handler) Register(r *httpx.Router) {
	r.HandleFunc(http.MethodGet, "/health/live", h.live)
	r.HandleFunc(http.MethodGet, "/health/ready", h.ready)
}

func (h *Handler) live(w http.ResponseWriter, _ *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": h.Service, "version": h.Version})
}

// CheckResult is the outcome of one check.
type CheckResult struct {
	Name     string `json:"name"`
	Status   string `json:"status"`
	Error    string `json:"error,omitempty"`
	Duration int64  `json:"duration_ms"`
}

// Report is the readiness response body.
type Report struct {
	Status  string        `json:"status"`
	Service string        `json:"service"`
	Checks  []CheckResult `json:"checks"`
}

// Evaluate runs all checks concurrently.
func (h *Handler) Evaluate(ctx context.Context) Report {
	timeout := h.Timeout
	if timeout == 0 {
		timeout = 2 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	res := make([]CheckResult, len(h.Checks))
	var wg sync.WaitGroup
	for i, c := range h.Checks {
		wg.Add(1)
		go func(i int, c Check) {
			defer wg.Done()
			start := time.Now()
			err := c.Run(ctx)
			r := CheckResult{Name: c.Name, Status: "ok", Duration: time.Since(start).Milliseconds()}
			if err != nil {
				r.Status, r.Error = "fail", err.Error()
			}
			res[i] = r
		}(i, c)
	}
	wg.Wait()
	rep := Report{Status: "ok", Service: h.Service, Checks: res}
	for _, r := range res {
		if r.Status != "ok" {
			rep.Status = "fail"
		}
	}
	return rep
}

func (h *Handler) ready(w http.ResponseWriter, r *http.Request) {
	rep := h.Evaluate(r.Context())
	status := http.StatusOK
	if rep.Status != "ok" {
		status = http.StatusServiceUnavailable
	}
	w.Header().Set("Cache-Control", "no-store")
	httpx.WriteJSON(w, status, rep)
}

// TCP checks that a TCP connection can be opened.
func TCP(name, addr string) Check {
	return Check{Name: name, Run: func(ctx context.Context) error {
		var d net.Dialer
		c, err := d.DialContext(ctx, "tcp", addr)
		if err != nil {
			return err
		}
		return c.Close()
	}}
}

// Valkey sends a RESP PING and expects +PONG (works with Valkey and Redis).
func Valkey(addr string) Check {
	return Check{Name: "valkey", Run: func(ctx context.Context) error {
		var d net.Dialer
		c, err := d.DialContext(ctx, "tcp", addr)
		if err != nil {
			return err
		}
		defer c.Close()
		if dl, ok := ctx.Deadline(); ok {
			_ = c.SetDeadline(dl)
		}
		if _, err := c.Write([]byte("*1\r\n$4\r\nPING\r\n")); err != nil {
			return err
		}
		line, err := bufio.NewReader(c).ReadString('\n')
		if err != nil {
			return err
		}
		if strings.TrimSpace(line) != "+PONG" {
			return fmt.Errorf("unexpected reply %q", strings.TrimSpace(line))
		}
		return nil
	}}
}

// HTTP checks that GET url answers 2xx (e.g. MinIO /minio/health/ready).
func HTTP(name, url string) Check {
	return Check{Name: name, Run: func(ctx context.Context) error {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return err
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}
		resp.Body.Close()
		if resp.StatusCode/100 != 2 {
			return fmt.Errorf("status %d", resp.StatusCode)
		}
		return nil
	}}
}
