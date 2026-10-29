import { useState, type FormEvent } from "react";
import { useLocation, useNavigate } from "react-router";
import { ApiError, Button, Card, TextField, ThemeToggle, messageFor, toLatinDigits } from "@vira/shared";
import { useAuth } from "../auth";

/** Two-step admin sign-in: password, then TOTP (ADR-016). No "remember me"; sessions are short-lived. */
export function LoginPage() {
  const { login, verifyMfa } = useAuth();
  const nav = useNavigate();
  const loc = useLocation();
  const [step, setStep] = useState<"password" | "totp">("password");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [mfaToken, setMfaToken] = useState("");
  const [code, setCode] = useState("");
  const [error, setError] = useState<string>();
  const [busy, setBusy] = useState(false);

  async function submit(e: FormEvent) {
    e.preventDefault();
    setError(undefined);
    setBusy(true);
    try {
      if (step === "password") {
        const r = await login(email.trim(), password);
        setMfaToken(r.mfa_token);
        setPassword("");
        setStep("totp");
      } else {
        await verifyMfa(mfaToken, toLatinDigits(code));
        const from = (loc.state as { from?: string } | null)?.from;
        nav(from && from !== "/login" ? from : "/", { replace: true });
      }
    } catch (err) {
      setError(err instanceof ApiError ? err.title : messageFor("internal_error"));
      if (err instanceof ApiError && step === "totp" && err.code !== "mfa_invalid") {
        setStep("password");
        setCode("");
      }
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="v-auth">
      <div style={{ position: "fixed", insetBlockStart: 16, insetInlineEnd: 16 }}><ThemeToggle /></div>
      <form className="v-auth__card" onSubmit={submit} noValidate>
        <div style={{ display: "flex", alignItems: "center", gap: 12 }}>
          <span className="v-brand__mark" aria-hidden>و</span>
          <div>
            <h1 className="v-auth__title">ورود به مدیریت ارشد ویرا</h1>
            <p className="v-auth__sub">{step === "password" ? "مرحلهٔ ۱ از ۲ — رمز عبور" : "مرحلهٔ ۲ از ۲ — کد تأیید دومرحله‌ای"}</p>
          </div>
        </div>
        <Card>
          <div style={{ display: "grid", gap: 16 }}>
            {step === "password" ? (
              <>
                <TextField label="ایمیل سازمانی" type="email" dir="ltr" autoComplete="username" required value={email}
                  onChange={(e) => setEmail(e.target.value)} />
                <TextField label="رمز عبور" type="password" dir="ltr" autoComplete="current-password" required value={password}
                  onChange={(e) => setPassword(e.target.value)} />
              </>
            ) : (
              <TextField label="کد ۶ رقمی برنامهٔ احراز هویت" inputMode="numeric" dir="ltr" autoComplete="one-time-code"
                maxLength={6} required value={code} autoFocus onChange={(e) => setCode(e.target.value.replace(/[^0-9۰-۹٠-٩]/g, ""))}
                hint="کد را از برنامهٔ Authenticator وارد کنید." />
            )}
            {error && <div className="v-field__error" role="alert">{error}</div>}
            <Button type="submit" loading={busy}
              disabled={step === "password" ? !email || !password : toLatinDigits(code).length !== 6}>
              {step === "password" ? "ادامه" : "ورود"}
            </Button>
          </div>
        </Card>
        <p className="v-auth__sub" style={{ font: "var(--type-caption)" }}>
          دسترسی فقط برای کارکنان مجاز. همهٔ ورودها و اقدام‌ها در گزارش ممیزی ثبت می‌شوند.
        </p>
      </form>
    </div>
  );
}
