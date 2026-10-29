// Package devsim implements the development-only provider simulators (doc 20 §3.1): an SMS inbox, a push
// inbox and a PSP internet-payment-gateway (IPG) simulator that follows the Iranian redirect flow
// (token → hosted payment page → browser callback → server verify). Nothing here moves money; the
// binary refuses to start in production and every page says so.
package devsim

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/NADER-CY/vira/backend/internal/platform/apperr"
	"github.com/NADER-CY/vira/backend/internal/platform/httpx"
)

const maxInbox = 200

// Message is an SMS or push notification captured by the simulator.
type Message struct {
	ID     string            `json:"id"`
	Kind   string            `json:"kind"` // sms | push
	To     string            `json:"to"`
	Title  string            `json:"title,omitempty"`
	Body   string            `json:"body"`
	Data   map[string]string `json:"data,omitempty"`
	SentAt time.Time         `json:"sent_at"`
}

// Payment states of the IPG simulator.
const (
	PayPending  = "pending"
	PayPaid     = "paid"     // payer approved; awaiting merchant verify
	PayFailed   = "failed"   // payer declined or cancelled
	PayVerified = "verified" // merchant verified (settled in the simulator only)
)

// Payment is one simulated IPG transaction.
type Payment struct {
	Token       string    `json:"token"`
	AmountRial  int64     `json:"amount_rial"`
	OrderRef    string    `json:"order_ref"`
	CallbackURL string    `json:"callback_url"`
	State       string    `json:"state"`
	RRN         string    `json:"rrn,omitempty"` // retrieval reference number
	CardMask    string    `json:"card_mask,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	VerifiedAt  time.Time `json:"verified_at,omitempty"`
}

// Sim holds simulator state in memory (development only; restarting clears it).
type Sim struct {
	mu       sync.Mutex
	messages []Message
	payments map[string]*Payment
	now      func() time.Time
}

// New returns an empty simulator.
func New() *Sim { return &Sim{payments: map[string]*Payment{}, now: time.Now} }

func randID(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func (s *Sim) record(m Message) Message {
	s.mu.Lock()
	defer s.mu.Unlock()
	m.ID, m.SentAt = randID(8), s.now()
	s.messages = append(s.messages, m)
	if len(s.messages) > maxInbox {
		s.messages = s.messages[len(s.messages)-maxInbox:]
	}
	return m
}

// Messages returns captured messages of a kind ("" = all), newest first.
func (s *Sim) Messages(kind string) []Message {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []Message{}
	for i := len(s.messages) - 1; i >= 0; i-- {
		if kind == "" || s.messages[i].Kind == kind {
			out = append(out, s.messages[i])
		}
	}
	return out
}

// Handler returns the simulator HTTP API and pages.
func (s *Sim) Handler() http.Handler {
	r := httpx.NewRouter()
	r.HandleFunc(http.MethodGet, "/", s.index)
	r.HandleFunc(http.MethodGet, "/health/live", func(w http.ResponseWriter, _ *http.Request) {
		httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	r.HandleFunc(http.MethodPost, "/sms/v1/send", s.sendSMS)
	r.HandleFunc(http.MethodGet, "/sms/v1/messages", func(w http.ResponseWriter, _ *http.Request) {
		httpx.WriteJSON(w, http.StatusOK, map[string]any{"data": s.Messages("sms")})
	})
	r.HandleFunc(http.MethodPost, "/push/v1/send", s.sendPush)
	r.HandleFunc(http.MethodGet, "/push/v1/messages", func(w http.ResponseWriter, _ *http.Request) {
		httpx.WriteJSON(w, http.StatusOK, map[string]any{"data": s.Messages("push")})
	})
	r.HandleFunc(http.MethodPost, "/psp/v1/token", s.pspToken)
	r.HandleFunc(http.MethodGet, "/psp/pay/{token}", s.pspPage)
	r.HandleFunc(http.MethodPost, "/psp/pay/{token}", s.pspDecide)
	r.HandleFunc(http.MethodPost, "/psp/v1/verify", s.pspVerify)
	r.HandleFunc(http.MethodPost, "/psp/v1/inquiry", s.pspInquiry)
	return r
}

func (s *Sim) sendSMS(w http.ResponseWriter, r *http.Request) {
	var in struct {
		To   string `json:"to"`
		Text string `json:"text"`
	}
	if err := httpx.DecodeJSON(r, &in); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if !validIranMobile(in.To) || strings.TrimSpace(in.Text) == "" {
		httpx.WriteError(w, r, apperr.New(apperr.ValidationFailed, "to must be an Iranian mobile number (09xxxxxxxxx or +989xxxxxxxxx) and text non-empty"))
		return
	}
	httpx.WriteJSON(w, http.StatusAccepted, s.record(Message{Kind: "sms", To: in.To, Body: in.Text}))
}

func (s *Sim) sendPush(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Token string            `json:"token"`
		Title string            `json:"title"`
		Body  string            `json:"body"`
		Data  map[string]string `json:"data"`
	}
	if err := httpx.DecodeJSON(r, &in); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if in.Token == "" || in.Body == "" {
		httpx.WriteError(w, r, apperr.New(apperr.ValidationFailed, "token and body are required"))
		return
	}
	httpx.WriteJSON(w, http.StatusAccepted, s.record(Message{Kind: "push", To: in.Token, Title: in.Title, Body: in.Body, Data: in.Data}))
}

// validIranMobile accepts 09xxxxxxxxx and +989xxxxxxxxx (ASCII digits).
func validIranMobile(p string) bool {
	switch {
	case strings.HasPrefix(p, "+989") && len(p) == 13:
		p = p[4:]
	case strings.HasPrefix(p, "09") && len(p) == 11:
		p = p[2:]
	default:
		return false
	}
	for _, c := range p {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

func (s *Sim) pspToken(w http.ResponseWriter, r *http.Request) {
	var in struct {
		AmountRial  int64  `json:"amount_rial"`
		OrderRef    string `json:"order_ref"`
		CallbackURL string `json:"callback_url"`
	}
	if err := httpx.DecodeJSON(r, &in); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	cb, err := url.Parse(in.CallbackURL)
	if in.AmountRial < 10_000 || in.OrderRef == "" || err != nil || (cb.Scheme != "http" && cb.Scheme != "https") || cb.Host == "" {
		httpx.WriteError(w, r, apperr.New(apperr.ValidationFailed, "amount_rial >= 10000, order_ref and an absolute http(s) callback_url are required"))
		return
	}
	p := &Payment{Token: randID(16), AmountRial: in.AmountRial, OrderRef: in.OrderRef, CallbackURL: in.CallbackURL,
		State: PayPending, CreatedAt: s.now()}
	s.mu.Lock()
	s.payments[p.Token] = p
	s.mu.Unlock()
	httpx.WriteJSON(w, http.StatusCreated, map[string]any{"token": p.Token, "payment_url": "/psp/pay/" + p.Token})
}

func (s *Sim) payment(token string) (Payment, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.payments[token]
	if !ok {
		return Payment{}, false
	}
	return *p, true
}

var payPage = template.Must(template.New("pay").Parse(`<!doctype html>
<html lang="fa" dir="rtl"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>درگاه پرداخت شبیه‌سازی‌شده</title>
<style>body{font-family:Vazirmatn,Tahoma,sans-serif;background:#f5f5f7;margin:0;padding:24px}
.card{max-width:420px;margin:auto;background:#fff;border-radius:16px;padding:24px;box-shadow:0 2px 12px #0001}
.warn{background:#fff3cd;color:#664d03;border-radius:10px;padding:12px;margin-bottom:16px;font-weight:700}
button{width:100%;padding:14px;border:0;border-radius:12px;font-size:16px;margin-top:10px;cursor:pointer}
.ok{background:#16a34a;color:#fff}.no{background:#e5e7eb}</style></head>
<body><div class="card">
<div class="warn">شبیه‌ساز توسعه — هیچ پول واقعی منتقل نمی‌شود. اطلاعات کارت وارد نکنید.</div>
<p>سفارش: <b>{{.OrderRef}}</b></p><p>مبلغ: <b>{{.AmountRial}}</b> ریال</p><p>وضعیت: {{.State}}</p>
{{if eq .State "pending"}}<form method="post"><button class="ok" name="decision" value="approve">تأیید پرداخت (شبیه‌سازی)</button>
<button class="no" name="decision" value="decline">انصراف</button></form>{{end}}
</div></body></html>`))

func (s *Sim) pspPage(w http.ResponseWriter, r *http.Request) {
	p, ok := s.payment(r.PathValue("token"))
	if !ok {
		httpx.WriteError(w, r, apperr.New(apperr.NotFound, "payment not found"))
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = payPage.Execute(w, p)
}

func (s *Sim) pspDecide(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		httpx.WriteError(w, r, apperr.New(apperr.ValidationFailed, "invalid form"))
		return
	}
	token := r.PathValue("token")
	s.mu.Lock()
	p, ok := s.payments[token]
	if ok && p.State == PayPending {
		if r.PostForm.Get("decision") == "approve" {
			p.State, p.RRN, p.CardMask = PayPaid, randID(6), "6037-99**-****-0000"
		} else {
			p.State = PayFailed
		}
	}
	var cp Payment
	if ok {
		cp = *p
	}
	s.mu.Unlock()
	if !ok {
		httpx.WriteError(w, r, apperr.New(apperr.NotFound, "payment not found"))
		return
	}
	cb, _ := url.Parse(cp.CallbackURL)
	q := cb.Query()
	q.Set("token", cp.Token)
	q.Set("status", map[bool]string{true: "OK", false: "NOK"}[cp.State == PayPaid])
	q.Set("order_ref", cp.OrderRef)
	cb.RawQuery = q.Encode()
	http.Redirect(w, r, cb.String(), http.StatusSeeOther)
}

func (s *Sim) pspVerify(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Token      string `json:"token"`
		AmountRial int64  `json:"amount_rial"`
	}
	if err := httpx.DecodeJSON(r, &in); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	s.mu.Lock()
	p, ok := s.payments[in.Token]
	var verr error
	switch {
	case !ok:
	case p.AmountRial != in.AmountRial:
		verr = apperr.New(apperr.ValidationFailed, "amount_rial does not match the payment")
	case p.State == PayPaid:
		p.State, p.VerifiedAt = PayVerified, s.now()
	case p.State == PayVerified: // idempotent: repeated verify returns the same result
	default:
		verr = apperr.New(apperr.Conflict, fmt.Sprintf("payment is %s", p.State))
	}
	var cp Payment
	if ok {
		cp = *p
	}
	s.mu.Unlock()
	switch {
	case !ok:
		httpx.WriteError(w, r, apperr.New(apperr.NotFound, "payment not found"))
	case verr != nil:
		httpx.WriteError(w, r, verr)
	default:
		httpx.WriteJSON(w, http.StatusOK, cp)
	}
}

func (s *Sim) pspInquiry(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Token string `json:"token"`
	}
	if err := httpx.DecodeJSON(r, &in); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	p, ok := s.payment(in.Token)
	if !ok {
		httpx.WriteError(w, r, apperr.New(apperr.NotFound, "payment not found"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, p)
}

var indexPage = template.Must(template.New("index").Parse(`<!doctype html>
<html lang="fa" dir="rtl"><head><meta charset="utf-8"><title>VIRA dev simulators</title>
<style>body{font-family:Vazirmatn,Tahoma,sans-serif;margin:24px}td,th{border-bottom:1px solid #ddd;padding:6px;text-align:right}
.warn{background:#fff3cd;padding:10px;border-radius:8px}</style></head><body>
<p class="warn">شبیه‌سازهای توسعه (پیامک، اعلان، درگاه پرداخت). فقط برای محیط توسعه — هیچ پیام یا پولی واقعاً ارسال نمی‌شود.</p>
<h2>پیامک‌ها و اعلان‌ها</h2><table><tr><th>نوع</th><th>گیرنده</th><th>متن</th><th>زمان</th></tr>
{{range .}}<tr><td>{{.Kind}}</td><td dir="ltr">{{.To}}</td><td>{{.Title}} {{.Body}}</td><td dir="ltr">{{.SentAt.Format "15:04:05"}}</td></tr>{{end}}
</table></body></html>`))

func (s *Sim) index(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = indexPage.Execute(w, s.Messages(""))
}
