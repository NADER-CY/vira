// Package objectstore is a minimal S3-compatible client (MinIO locally, provider object storage when deployed)
// with AWS Signature Version 4 request signing. Only what media needs: put, get, ensure bucket.
package objectstore

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

// Store is what the media module uses; tests use Memory.
type Store interface {
	Put(ctx context.Context, key, contentType string, body []byte) error
	Get(ctx context.Context, key string) (io.ReadCloser, string, error)
}

// ErrNotFound is returned for missing objects.
var ErrNotFound = errors.New("objectstore: not found")

// S3 talks to an S3-compatible endpoint with path-style addressing.
type S3 struct {
	Endpoint  string // e.g. http://minio:9000
	Region    string
	Bucket    string
	AccessKey string
	SecretKey string
	Client    *http.Client
	Now       func() time.Time
}

func (s *S3) client() *http.Client {
	if s.Client != nil {
		return s.Client
	}
	return &http.Client{Timeout: 30 * time.Second}
}

func (s *S3) objectURL(key string) string {
	return strings.TrimRight(s.Endpoint, "/") + "/" + s.Bucket + "/" + escapePath(key)
}

// EnsureBucket creates the bucket if it does not exist (idempotent).
func (s *S3) EnsureBucket(ctx context.Context) error {
	u := strings.TrimRight(s.Endpoint, "/") + "/" + s.Bucket
	resp, err := s.do(ctx, http.MethodHead, u, "", nil)
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		return nil
	}
	resp, err = s.do(ctx, http.MethodPut, u, "", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusConflict {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("objectstore: create bucket: %d %s", resp.StatusCode, b)
	}
	return nil
}

// Put stores an object.
func (s *S3) Put(ctx context.Context, key, contentType string, body []byte) error {
	resp, err := s.do(ctx, http.MethodPut, s.objectURL(key), contentType, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("objectstore: put: %d %s", resp.StatusCode, b)
	}
	return nil
}

// Get streams an object; the caller closes the body.
func (s *S3) Get(ctx context.Context, key string) (io.ReadCloser, string, error) {
	resp, err := s.do(ctx, http.MethodGet, s.objectURL(key), "", nil)
	if err != nil {
		return nil, "", err
	}
	switch resp.StatusCode {
	case http.StatusOK:
		return resp.Body, resp.Header.Get("Content-Type"), nil
	case http.StatusNotFound:
		resp.Body.Close()
		return nil, "", ErrNotFound
	default:
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		resp.Body.Close()
		return nil, "", fmt.Errorf("objectstore: get: %d %s", resp.StatusCode, b)
	}
}

func (s *S3) do(ctx context.Context, method, rawURL, contentType string, body []byte) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, rawURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	if body == nil {
		req.Body = http.NoBody
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	now := time.Now
	if s.Now != nil {
		now = s.Now
	}
	Sign(req, body, s.AccessKey, s.SecretKey, s.Region, "s3", now().UTC())
	return s.client().Do(req)
}

// Sign adds AWS Signature Version 4 headers to req (payload hash included).
func Sign(req *http.Request, body []byte, accessKey, secretKey, region, service string, t time.Time) {
	amzDate := t.Format("20060102T150405Z")
	day := t.Format("20060102")
	payloadHash := sha256Hex(body)
	req.Header.Set("X-Amz-Date", amzDate)
	req.Header.Set("X-Amz-Content-Sha256", payloadHash)
	if req.Host == "" {
		req.Host = req.URL.Host
	}

	names := []string{"host"}
	for k := range req.Header {
		lk := strings.ToLower(k)
		if lk == "content-type" || strings.HasPrefix(lk, "x-amz-") {
			names = append(names, lk)
		}
	}
	sort.Strings(names)
	var canonHeaders strings.Builder
	for _, n := range names {
		v := req.Host
		if n != "host" {
			v = strings.TrimSpace(req.Header.Get(n))
		}
		canonHeaders.WriteString(n + ":" + v + "\n")
	}
	signed := strings.Join(names, ";")
	canonical := strings.Join([]string{req.Method, escapePath(req.URL.EscapedPath()), canonicalQuery(req.URL.Query()),
		canonHeaders.String(), signed, payloadHash}, "\n")
	scope := day + "/" + region + "/" + service + "/aws4_request"
	toSign := "AWS4-HMAC-SHA256\n" + amzDate + "\n" + scope + "\n" + sha256Hex([]byte(canonical))
	k := hmacSHA256([]byte("AWS4"+secretKey), day)
	k = hmacSHA256(k, region)
	k = hmacSHA256(k, service)
	k = hmacSHA256(k, "aws4_request")
	sig := hex.EncodeToString(hmacSHA256(k, toSign))
	req.Header.Set("Authorization", "AWS4-HMAC-SHA256 Credential="+accessKey+"/"+scope+", SignedHeaders="+signed+", Signature="+sig)
}

func canonicalQuery(q url.Values) string {
	keys := make([]string, 0, len(q))
	for k := range q {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var parts []string
	for _, k := range keys {
		vs := q[k]
		sort.Strings(vs)
		for _, v := range vs {
			parts = append(parts, uriEncode(k, true)+"="+uriEncode(v, true))
		}
	}
	return strings.Join(parts, "&")
}

// escapePath URI-encodes each path segment (keeps "/").
func escapePath(p string) string {
	if u, err := url.PathUnescape(p); err == nil {
		p = u
	}
	segs := strings.Split(p, "/")
	for i, s := range segs {
		segs[i] = uriEncode(s, true)
	}
	return strings.Join(segs, "/")
}

func uriEncode(s string, encodeSlash bool) string {
	var b strings.Builder
	for _, c := range []byte(s) {
		switch {
		case c >= 'A' && c <= 'Z', c >= 'a' && c <= 'z', c >= '0' && c <= '9', c == '-', c == '_', c == '.', c == '~':
			b.WriteByte(c)
		case c == '/' && !encodeSlash:
			b.WriteByte(c)
		default:
			fmt.Fprintf(&b, "%%%02X", c)
		}
	}
	return b.String()
}

func sha256Hex(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

func hmacSHA256(key []byte, data string) []byte {
	m := hmac.New(sha256.New, key)
	m.Write([]byte(data))
	return m.Sum(nil)
}

// Memory is an in-process Store for tests.
type Memory struct {
	mu   sync.Mutex
	objs map[string]memObj
}

type memObj struct {
	ct   string
	data []byte
}

// Put stores an object.
func (m *Memory) Put(_ context.Context, key, contentType string, body []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.objs == nil {
		m.objs = map[string]memObj{}
	}
	m.objs[key] = memObj{contentType, append([]byte(nil), body...)}
	return nil
}

// Get returns an object.
func (m *Memory) Get(_ context.Context, key string) (io.ReadCloser, string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	o, ok := m.objs[key]
	if !ok {
		return nil, "", ErrNotFound
	}
	return io.NopCloser(bytes.NewReader(o.data)), o.ct, nil
}

// Len reports the number of stored objects.
func (m *Memory) Len() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.objs)
}
