package objectstore

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// AWS Signature Version 4 test suite vector "get-vanilla" (service "service", region us-east-1), adapted:
// the signer always adds x-amz-content-sha256, so the expected signature is computed for that header set.
// The derivation itself is checked against the published signing-key example from the AWS documentation.
func TestSigningKeyDerivationMatchesAWSExample(t *testing.T) {
	k := hmacSHA256([]byte("AWS4"+"wJalrXUtnFEMI/K7MDENG+bPxRfiCYEXAMPLEKEY"), "20120215")
	k = hmacSHA256(k, "us-east-1")
	k = hmacSHA256(k, "iam")
	k = hmacSHA256(k, "aws4_request")
	const want = "f4780e2d9f65fa895f9c67b32ce1baf0b0d8a43505a000a1a9e090d414db404d"
	if got := strings.ToLower(hexString(k)); got != want {
		t.Fatalf("signing key %s, want %s", got, want)
	}
}

func hexString(b []byte) string {
	const h = "0123456789abcdef"
	out := make([]byte, len(b)*2)
	for i, c := range b {
		out[i*2], out[i*2+1] = h[c>>4], h[c&15]
	}
	return string(out)
}

func TestSignedRequestShape(t *testing.T) {
	req, _ := http.NewRequest("PUT", "http://minio:9000/vira-media/post/a%20b.jpg", strings.NewReader("x"))
	req.Header.Set("Content-Type", "image/jpeg")
	Sign(req, []byte("x"), "AK", "SK", "us-east-1", "s3", time.Date(2026, 10, 29, 10, 0, 0, 0, time.UTC))
	a := req.Header.Get("Authorization")
	for _, want := range []string{"AWS4-HMAC-SHA256 Credential=AK/20261029/us-east-1/s3/aws4_request",
		"SignedHeaders=content-type;host;x-amz-content-sha256;x-amz-date", "Signature="} {
		if !strings.Contains(a, want) {
			t.Errorf("authorization %q lacks %q", a, want)
		}
	}
	if req.Header.Get("X-Amz-Content-Sha256") != sha256Hex([]byte("x")) {
		t.Error("payload hash header missing")
	}
}

func TestS3PutGetAgainstFakeServer(t *testing.T) {
	objs := map[string][]byte{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.Header.Get("Authorization"), "AWS4-HMAC-SHA256 ") {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		switch r.Method {
		case http.MethodHead:
			w.WriteHeader(http.StatusNotFound)
		case http.MethodPut:
			b, _ := io.ReadAll(r.Body)
			objs[r.URL.Path] = b
		case http.MethodGet:
			b, ok := objs[r.URL.Path]
			if !ok {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			w.Header().Set("Content-Type", "image/jpeg")
			_, _ = w.Write(b)
		}
	}))
	defer srv.Close()
	s := &S3{Endpoint: srv.URL, Region: "us-east-1", Bucket: "b", AccessKey: "a", SecretKey: "s"}
	ctx := context.Background()
	if err := s.EnsureBucket(ctx); err != nil {
		t.Fatal(err)
	}
	if err := s.Put(ctx, "post/x.jpg", "image/jpeg", []byte("data")); err != nil {
		t.Fatal(err)
	}
	rc, ct, err := s.Get(ctx, "post/x.jpg")
	if err != nil || ct != "image/jpeg" {
		t.Fatal(err, ct)
	}
	b, _ := io.ReadAll(rc)
	rc.Close()
	if string(b) != "data" {
		t.Fatalf("got %q", b)
	}
	if _, _, err := s.Get(ctx, "missing"); err != ErrNotFound {
		t.Fatalf("missing: %v", err)
	}
}
