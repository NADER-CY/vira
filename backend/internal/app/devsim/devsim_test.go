package devsim

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func do(t *testing.T, h http.Handler, method, path, body, ctype string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if ctype != "" {
		req.Header.Set("Content-Type", ctype)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestSMSInbox(t *testing.T) {
	s := New()
	h := s.Handler()
	if rec := do(t, h, "POST", "/sms/v1/send", `{"to":"09121234567","text":"کد ورود: 123456"}`, "application/json"); rec.Code != 202 {
		t.Fatalf("send: %d %s", rec.Code, rec.Body)
	}
	if rec := do(t, h, "POST", "/sms/v1/send", `{"to":"12345","text":"x"}`, "application/json"); rec.Code != 422 {
		t.Fatalf("invalid number accepted: %d", rec.Code)
	}
	if got := s.Messages("sms"); len(got) != 1 || got[0].To != "09121234567" {
		t.Fatalf("inbox = %+v", got)
	}
}

func TestPSPFlowApproveVerifyIdempotent(t *testing.T) {
	s := New()
	h := s.Handler()
	rec := do(t, h, "POST", "/psp/v1/token", `{"amount_rial":1500000,"order_ref":"ORD-1","callback_url":"http://localhost:8080/cb?x=1"}`, "application/json")
	if rec.Code != 201 {
		t.Fatalf("token: %d %s", rec.Code, rec.Body)
	}
	var token string
	for k := range s.payments {
		token = k
	}
	if page := do(t, h, "GET", "/psp/pay/"+token, "", ""); page.Code != 200 || !strings.Contains(page.Body.String(), "هیچ پول واقعی") {
		t.Fatalf("payment page must carry the no-real-money banner: %d", page.Code)
	}
	// Verify before approval is refused.
	if v := do(t, h, "POST", "/psp/v1/verify", `{"token":"`+token+`","amount_rial":1500000}`, "application/json"); v.Code != 409 {
		t.Fatalf("verify before approval: %d", v.Code)
	}
	cb := do(t, h, "POST", "/psp/pay/"+token, "decision=approve", "application/x-www-form-urlencoded")
	if cb.Code != http.StatusSeeOther {
		t.Fatalf("decide: %d", cb.Code)
	}
	loc, _ := url.Parse(cb.Header().Get("Location"))
	if loc.Query().Get("status") != "OK" || loc.Query().Get("x") != "1" || loc.Query().Get("token") != token {
		t.Fatalf("callback = %s", loc)
	}
	if v := do(t, h, "POST", "/psp/v1/verify", `{"token":"`+token+`","amount_rial":999999}`, "application/json"); v.Code != 422 {
		t.Fatalf("amount mismatch must fail: %d", v.Code)
	}
	for i := 0; i < 2; i++ {
		v := do(t, h, "POST", "/psp/v1/verify", `{"token":"`+token+`","amount_rial":1500000}`, "application/json")
		if v.Code != 200 || !strings.Contains(v.Body.String(), `"state":"verified"`) {
			t.Fatalf("verify #%d: %d %s", i, v.Code, v.Body)
		}
	}
	// A decided payment cannot be decided again.
	do(t, h, "POST", "/psp/pay/"+token, "decision=decline", "application/x-www-form-urlencoded")
	if p, _ := s.payment(token); p.State != PayVerified {
		t.Fatalf("state changed after verify: %s", p.State)
	}
}

func TestPSPDecline(t *testing.T) {
	s := New()
	h := s.Handler()
	do(t, h, "POST", "/psp/v1/token", `{"amount_rial":20000,"order_ref":"ORD-2","callback_url":"https://app.example/cb"}`, "application/json")
	var token string
	for k := range s.payments {
		token = k
	}
	cb := do(t, h, "POST", "/psp/pay/"+token, "decision=decline", "application/x-www-form-urlencoded")
	loc, _ := url.Parse(cb.Header().Get("Location"))
	if loc.Query().Get("status") != "NOK" {
		t.Fatalf("decline callback = %s", loc)
	}
	if v := do(t, h, "POST", "/psp/v1/verify", `{"token":"`+token+`","amount_rial":20000}`, "application/json"); v.Code != 409 {
		t.Fatalf("verify declined: %d", v.Code)
	}
}

func TestIranMobileValidation(t *testing.T) {
	for in, want := range map[string]bool{"09121234567": true, "+989121234567": true, "0912123456": false, "۰۹۱۲۱۲۳۴۵۶۷": false, "+98912123456a": false} {
		if got := validIranMobile(in); got != want {
			t.Errorf("%q: got %v", in, got)
		}
	}
}
