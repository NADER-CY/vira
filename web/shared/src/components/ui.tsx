import { useId, type ButtonHTMLAttributes, type InputHTMLAttributes, type ReactNode } from "react";
import { Icon, type IconName } from "./Icon";
import { useTheme } from "../theme";

type Variant = "primary" | "secondary" | "ghost" | "danger";

export function Button({ variant = "primary", icon, loading, children, className, ...rest }:
  ButtonHTMLAttributes<HTMLButtonElement> & { variant?: Variant; icon?: IconName; loading?: boolean }) {
  return (
    <button
      {...rest}
      className={`v-btn v-btn--${variant} ${children ? "" : "v-btn--icon"} ${className ?? ""}`.trim()}
      disabled={rest.disabled || loading}
      aria-busy={loading || undefined}
    >
      {loading ? <span className="v-spinner" style={{ inlineSize: 18, blockSize: 18, borderWidth: 2 }} aria-hidden /> : icon && <Icon name={icon} />}
      {children}
    </button>
  );
}

export function Card({ title, actions, children, className }: { title?: ReactNode; actions?: ReactNode; children: ReactNode; className?: string }) {
  return (
    <section className={`v-card ${className ?? ""}`.trim()}>
      {(title || actions) && (
        <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", gap: 12 }}>
          {title && <h2 className="v-card__title">{title}</h2>}
          {actions}
        </div>
      )}
      {children}
    </section>
  );
}

export type Tone = "neutral" | "success" | "warning" | "danger" | "info" | "brand";

export function Badge({ tone = "neutral", icon, children }: { tone?: Tone; icon?: IconName; children: ReactNode }) {
  return <span className={`v-badge v-badge--${tone}`}>{icon && <Icon name={icon} />}{children}</span>;
}

export function Spinner({ label = "در حال بارگذاری" }: { label?: string }) {
  return <span role="status" aria-live="polite"><span className="v-spinner" aria-hidden /><span className="visually-hidden">{label}</span></span>;
}

export function Skeleton({ width = "100%", height = 16 }: { width?: number | string; height?: number | string }) {
  return <span className="v-skeleton" style={{ display: "block", inlineSize: width, blockSize: height }} aria-hidden />;
}

export function TextField({ label, error, hint, ...rest }: InputHTMLAttributes<HTMLInputElement> & { label: string; error?: string; hint?: string }) {
  const id = useId();
  const errId = `${id}-err`;
  return (
    <div className="v-field">
      <label htmlFor={id}>{label}</label>
      <input id={id} className="v-input" aria-invalid={error ? true : undefined} aria-describedby={error ? errId : undefined} {...rest} />
      {hint && !error && <span className="v-field__hint" style={{ font: "var(--type-caption)", color: "var(--color-on-surface-variant)" }}>{hint}</span>}
      {error && <span id={errId} className="v-field__error" role="alert">{error}</span>}
    </div>
  );
}

export function Switch({ checked, onChange, disabled, label }: { checked: boolean; onChange?: (v: boolean) => void; disabled?: boolean; label: string }) {
  return (
    <button type="button" role="switch" className="v-switch" aria-checked={checked} aria-label={label} disabled={disabled}
      onClick={() => onChange?.(!checked)} />
  );
}

export function ThemeToggle() {
  const { resolved, toggle } = useTheme();
  const dark = resolved === "dark";
  return (
    <Button variant="ghost" icon={dark ? "light_mode" : "dark_mode"} onClick={toggle}
      aria-label={dark ? "پوستهٔ روشن" : "پوستهٔ تیره"} title={dark ? "پوستهٔ روشن" : "پوستهٔ تیره"} />
  );
}

export function PageHeader({ title, subtitle, actions }: { title: ReactNode; subtitle?: ReactNode; actions?: ReactNode }) {
  return (
    <header className="v-page-header">
      <div>
        <h1>{title}</h1>
        {subtitle && <p>{subtitle}</p>}
      </div>
      {actions && <div style={{ display: "flex", gap: 8 }}>{actions}</div>}
    </header>
  );
}
