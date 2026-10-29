// Package apperr defines the stable error codes of the VIRA API (doc 07 §1.1) and the
// application error type that the HTTP layer renders as RFC 9457 problem+json.
package apperr

import (
	"errors"
	"net/http"
)

// Code is a stable, machine-readable error code. Clients branch on Code, never on text.
type Code string

const (
	ValidationFailed     Code = "validation_failed"
	Unauthenticated      Code = "unauthenticated"
	Forbidden            Code = "forbidden"
	NotFound             Code = "not_found"
	MethodNotAllowed     Code = "method_not_allowed"
	Conflict             Code = "conflict"
	IdempotencyConflict  Code = "idempotency_conflict"
	RequestInProgress    Code = "request_in_progress"
	PreconditionFailed   Code = "precondition_failed"
	PreconditionRequired Code = "precondition_required"
	RateLimited          Code = "rate_limited"
	FeatureDisabled      Code = "feature_disabled"
	FeatureLocked        Code = "feature_locked"
	AccountRestricted    Code = "account_restricted"
	MFARequired          Code = "mfa_required"
	MFAInvalid           Code = "mfa_invalid"
	Maintenance          Code = "maintenance"
	UpgradeRequired      Code = "upgrade_required"
	Internal             Code = "internal_error"
	Unavailable          Code = "service_unavailable"
)

type meta struct {
	status int
	title  string // Persian, user-facing (doc 07 §1: messages in Persian plus a stable code)
}

var registry = map[Code]meta{
	ValidationFailed:     {http.StatusUnprocessableEntity, "اطلاعات ارسال‌شده معتبر نیست."},
	Unauthenticated:      {http.StatusUnauthorized, "لطفاً دوباره وارد شوید."},
	Forbidden:            {http.StatusForbidden, "اجازهٔ انجام این کار را ندارید."},
	NotFound:             {http.StatusNotFound, "موردی پیدا نشد."},
	MethodNotAllowed:     {http.StatusMethodNotAllowed, "این عملیات پشتیبانی نمی‌شود."},
	Conflict:             {http.StatusConflict, "این درخواست با وضعیت فعلی سازگار نیست."},
	IdempotencyConflict:  {http.StatusConflict, "این کلید درخواست قبلاً با محتوای دیگری استفاده شده است."},
	RequestInProgress:    {http.StatusConflict, "درخواست قبلی شما هنوز در حال انجام است."},
	PreconditionFailed:   {http.StatusPreconditionFailed, "این مورد در این فاصله تغییر کرده است؛ صفحه را تازه کنید."},
	PreconditionRequired: {http.StatusPreconditionRequired, "برای ویرایش، نسخهٔ فعلی (If-Match) لازم است."},
	RateLimited:          {http.StatusTooManyRequests, "تعداد درخواست‌ها زیاد است؛ کمی بعد دوباره تلاش کنید."},
	FeatureDisabled:      {http.StatusForbidden, "این قابلیت در حال حاضر فعال نیست."},
	FeatureLocked:        {http.StatusConflict, "این قابلیت تا تأیید مجوزهای لازم قفل است."},
	AccountRestricted:    {http.StatusForbidden, "حساب شما محدود شده است."},
	MFARequired:          {http.StatusUnauthorized, "کد تأیید دومرحله‌ای لازم است."},
	MFAInvalid:           {http.StatusUnauthorized, "کد تأیید نادرست یا منقضی است."},
	Maintenance:          {http.StatusServiceUnavailable, "سرویس در حال به‌روزرسانی است."},
	UpgradeRequired:      {http.StatusUpgradeRequired, "لطفاً برنامه را به‌روزرسانی کنید."},
	Internal:             {http.StatusInternalServerError, "خطای غیرمنتظره‌ای رخ داد."},
	Unavailable:          {http.StatusServiceUnavailable, "سرویس موقتاً در دسترس نیست."},
}

// FieldError describes one invalid input field.
type FieldError struct {
	Field  string `json:"field"`
	Code   string `json:"code"`
	Detail string `json:"detail,omitempty"`
}

// Error is an application error with a stable code.
type Error struct {
	Code   Code
	Detail string
	Fields []FieldError
	Extra  map[string]any // extension members (e.g. required_level)
	cause  error
}

func (e *Error) Error() string {
	if e.Detail != "" {
		return string(e.Code) + ": " + e.Detail
	}
	return string(e.Code)
}

func (e *Error) Unwrap() error { return e.cause }

// Status is the HTTP status for the error's code.
func (e *Error) Status() int { return Status(e.Code) }

// New creates an application error.
func New(code Code, detail string) *Error { return &Error{Code: code, Detail: detail} }

// Wrap creates an application error that keeps the underlying cause for logs (never sent to clients).
func Wrap(code Code, cause error) *Error { return &Error{Code: code, cause: cause} }

// Validation creates a validation_failed error with field errors.
func Validation(fields ...FieldError) *Error {
	return &Error{Code: ValidationFailed, Fields: fields}
}

// Status returns the HTTP status for a code (500 for unknown codes).
func Status(c Code) int {
	if m, ok := registry[c]; ok {
		return m.status
	}
	return http.StatusInternalServerError
}

// Title returns the Persian user-facing title for a code.
func Title(c Code) string {
	if m, ok := registry[c]; ok {
		return m.title
	}
	return registry[Internal].title
}

// Codes lists every registered code (used by contract tests).
func Codes() []Code {
	out := make([]Code, 0, len(registry))
	for c := range registry {
		out = append(out, c)
	}
	return out
}

// As extracts an *Error from err. Unknown errors become internal_error (cause kept for logging).
func As(err error) *Error {
	var e *Error
	if errors.As(err, &e) {
		return e
	}
	return Wrap(Internal, err)
}
