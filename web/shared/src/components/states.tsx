// UI state catalogue (doc 09 §5): loading, empty, error, feature disabled / locked.
import type { ReactNode } from "react";
import { ApiError } from "../lib/api";
import { messageFor } from "../lib/messages";
import { Icon, type IconName } from "./Icon";
import { Button, Skeleton } from "./ui";

export function LoadingState({ rows = 4 }: { rows?: number }) {
  return (
    <div role="status" aria-live="polite" aria-label="در حال بارگذاری" style={{ display: "grid", gap: 12 }}>
      {Array.from({ length: rows }, (_, i) => <Skeleton key={i} height={i === 0 ? 28 : 18} width={i === 0 ? "40%" : "100%"} />)}
    </div>
  );
}

export function EmptyState({ icon = "inbox", title, description, action }: { icon?: IconName; title: string; description?: string; action?: ReactNode }) {
  return (
    <div className="v-state">
      <span className="v-state__icon"><Icon name={icon} /></span>
      <p className="v-state__title">{title}</p>
      {description && <p>{description}</p>}
      {action}
    </div>
  );
}

export function ErrorState({ error, onRetry }: { error: unknown; onRetry?: () => void }) {
  const e = error instanceof ApiError ? error : null;
  const title = e ? e.title : messageFor("internal_error");
  return (
    <div className="v-state" role="alert">
      <span className="v-state__icon" style={{ background: "var(--color-danger-container)", color: "var(--color-danger)" }}><Icon name="error" /></span>
      <p className="v-state__title">{title}</p>
      {e?.traceId && <p style={{ font: "var(--type-caption)" }}>کد پیگیری: <span className="ltr">{e.traceId}</span></p>}
      {onRetry && <Button variant="secondary" icon="refresh" onClick={onRetry}>تلاش دوباره</Button>}
    </div>
  );
}

/** Shown where a capability exists in the roadmap but is not active yet (doc 09 §5 "Feature disabled"). */
export function FeatureDisabledState({ title, phase, locked }: { title: string; phase: string; locked?: boolean }) {
  return (
    <EmptyState
      icon={locked ? "lock" : "flag"}
      title={title}
      description={locked
        ? `این قابلیت تا عبور از دروازهٔ ${phase} (تأیید حقوقی، امنیتی و تأمین‌کننده) قفل است.`
        : `این بخش در ${phase} پیاده‌سازی می‌شود.`}
    />
  );
}
