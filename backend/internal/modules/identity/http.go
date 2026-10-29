package identity

import (
	"net"
	"net/http"
	"time"

	"github.com/NADER-CY/vira/backend/internal/modules/identity/contract"
	"github.com/NADER-CY/vira/backend/internal/platform/apperr"
	"github.com/NADER-CY/vira/backend/internal/platform/auth"
	"github.com/NADER-CY/vira/backend/internal/platform/httpx"
)

// PoPHeader carries the device's proof-of-possession signature on refresh: "<unix-seconds>.<base64url DER sig>".
const PoPHeader = "X-Vira-PoP"

// DeviceHeader identifies the client installation (used for rate limits before sign-in).
const DeviceHeader = "X-Device-Id"

// Meta extracts request attributes for security events.
func Meta(r *http.Request) RequestMeta {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	ip := ""
	if parsed := net.ParseIP(host); parsed != nil {
		ip = parsed.String()
	}
	dev := r.Header.Get(DeviceHeader)
	if len(dev) > 128 {
		dev = dev[:128]
	}
	return RequestMeta{IP: ip, UserAgent: r.UserAgent(), TraceID: httpx.TraceID(r.Context()), DeviceID: dev}
}

// RequireSession is the authentication middleware for signed-in routes: a valid access token AND a live
// server-side session. It records the account status for write checks (contract.CanWrite).
func (s *Service) RequireSession(v *auth.Verifier) httpx.Middleware {
	authn := v.Middleware(true)
	return func(next http.Handler) http.Handler {
		check := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			p, _ := auth.FromContext(r.Context())
			st, err := s.CheckSession(r.Context(), p.Subject, p.SessionID)
			if err != nil {
				w.Header().Set("WWW-Authenticate", `Bearer realm="vira", error="invalid_token"`)
				httpx.WriteError(w, r, err)
				return
			}
			next.ServeHTTP(w, r.WithContext(contract.WithAccountStatus(r.Context(), st.Status)))
		})
		return authn(check)
	}
}

// Routes registers the identity endpoints (doc 07 §2 Auth, Me).
func (s *Service) Routes(r *httpx.Router, v *auth.Verifier) {
	sess := s.RequireSession(v)
	r.HandleFunc(http.MethodPost, "/v1/auth/otp", s.hRequestOTP)
	r.HandleFunc(http.MethodPost, "/v1/auth/otp/verify", s.hVerifyOTP)
	r.HandleFunc(http.MethodPost, "/v1/auth/token/refresh", s.hRefresh)
	r.HandleFunc(http.MethodPost, "/v1/auth/recovery", s.hRecovery)
	r.HandleFunc(http.MethodPost, "/v1/auth/logout", s.hLogout, sess)
	r.HandleFunc(http.MethodGet, "/v1/me/account", s.hAccount, sess)
	r.HandleFunc(http.MethodGet, "/v1/me/sessions", s.hSessions, sess)
	r.HandleFunc(http.MethodDelete, "/v1/me/sessions/{id}", s.hRevokeSession, sess)
	r.HandleFunc(http.MethodPost, "/v1/me/sessions/revoke-others", s.hRevokeOthers, sess)
	r.HandleFunc(http.MethodPost, "/v1/me/deletion", s.hDeletion, sess)
}

func principal(r *http.Request) auth.Principal {
	p, _ := auth.FromContext(r.Context())
	return p
}

func (s *Service) hRequestOTP(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Phone   string `json:"phone"`
		Purpose string `json:"purpose"`
	}
	if err := httpx.DecodeJSON(r, &in); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if in.Purpose != "" && in.Purpose != "login" {
		httpx.WriteError(w, r, apperr.Validation(apperr.FieldError{Field: "purpose", Code: "invalid"}))
		return
	}
	out, err := s.RequestOTP(r.Context(), in.Phone, Meta(r))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	httpx.WriteJSON(w, http.StatusAccepted, out)
}

func (s *Service) hVerifyOTP(w http.ResponseWriter, r *http.Request) {
	var in struct {
		ChallengeID   string     `json:"challenge_id"`
		Code          string     `json:"code"`
		Device        DeviceInfo `json:"device"`
		IntendsToSell bool       `json:"intends_to_sell"`
		Consents      Consents   `json:"consents"`
	}
	if err := httpx.DecodeJSON(r, &in); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	out, err := s.VerifyOTP(r.Context(), in.ChallengeID, in.Code, in.Device, in.IntendsToSell, in.Consents, Meta(r))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	httpx.WriteJSON(w, http.StatusOK, out)
}

func (s *Service) hRefresh(w http.ResponseWriter, r *http.Request) {
	var in struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := httpx.DecodeJSON(r, &in); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	out, err := s.Refresh(r.Context(), in.RefreshToken, r.Header.Get(PoPHeader), Meta(r))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	httpx.WriteJSON(w, http.StatusOK, out)
}

func (s *Service) hRecovery(w http.ResponseWriter, r *http.Request) {
	var in struct {
		OldPhone string `json:"old_phone"`
		NewPhone string `json:"new_phone"`
		Note     string `json:"note"`
	}
	if err := httpx.DecodeJSON(r, &in); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if err := s.RequestRecovery(r.Context(), in.OldPhone, in.NewPhone, in.Note, Meta(r)); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusAccepted, map[string]string{"status": "received"})
}

func (s *Service) hLogout(w http.ResponseWriter, r *http.Request) {
	p := principal(r)
	if err := s.Logout(r.Context(), p.Subject, p.SessionID, Meta(r)); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Service) hAccount(w http.ResponseWriter, r *http.Request) {
	a, err := s.AccountOf(r.Context(), principal(r).Subject)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	httpx.WriteJSON(w, http.StatusOK, a)
}

func (s *Service) hSessions(w http.ResponseWriter, r *http.Request) {
	p := principal(r)
	list, err := s.Sessions(r.Context(), p.Subject, p.SessionID)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"data": list})
}

func (s *Service) hRevokeSession(w http.ResponseWriter, r *http.Request) {
	if err := s.RevokeSession(r.Context(), principal(r).Subject, r.PathValue("id"), Meta(r)); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Service) hRevokeOthers(w http.ResponseWriter, r *http.Request) {
	p := principal(r)
	if err := s.RevokeOtherSessions(r.Context(), p.Subject, p.SessionID, Meta(r)); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Service) hDeletion(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Confirm bool `json:"confirm"`
	}
	if err := httpx.DecodeJSON(r, &in); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if !in.Confirm {
		httpx.WriteError(w, r, apperr.Validation(apperr.FieldError{Field: "confirm", Code: "required"}))
		return
	}
	eff, err := s.RequestDeletion(r.Context(), principal(r).Subject, Meta(r))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusAccepted, map[string]any{"status": contract.StatusDeletionPending, "effective_at": eff.UTC().Format(time.RFC3339)})
}
