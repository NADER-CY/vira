// Persian user-facing messages for API error codes (doc 07 §1). Server titles are already Persian; these are
// the client fallbacks, used when no title is available (network errors, unknown codes).
export const errorMessages: Record<string, string> = {
  validation_failed: "اطلاعات واردشده معتبر نیست.",
  unauthenticated: "لطفاً دوباره وارد شوید.",
  forbidden: "دسترسی به این بخش برای شما مجاز نیست.",
  not_found: "مورد درخواستی پیدا نشد.",
  conflict: "این مورد هم‌زمان تغییر کرده است. صفحه را تازه کنید.",
  precondition_failed: "این مورد هم‌زمان تغییر کرده است. صفحه را تازه کنید.",
  rate_limited: "تعداد درخواست‌ها زیاد است. کمی بعد دوباره تلاش کنید.",
  feature_disabled: "این قابلیت هنوز فعال نشده است.",
  feature_locked: "این قابلیت تا تأیید مراحل قانونی و امنیتی قفل است.",
  mfa_required: "تأیید دومرحله‌ای لازم است.",
  mfa_invalid: "کد تأیید نادرست یا منقضی است.",
  maintenance: "سامانه در حال به‌روزرسانی است.",
  service_unavailable: "سرویس موقتاً در دسترس نیست.",
  internal_error: "خطای غیرمنتظره رخ داد. دوباره تلاش کنید.",
  network_error: "اتصال به سرور برقرار نشد. اتصال اینترنت را بررسی کنید.",
  timeout: "پاسخ سرور طول کشید. دوباره تلاش کنید.",
};

export function messageFor(code: string, fallback?: string): string {
  return errorMessages[code] ?? fallback ?? errorMessages.internal_error!;
}
