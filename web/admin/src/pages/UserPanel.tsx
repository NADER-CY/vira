import { useState } from "react";
import {
  ApiError, Badge, Button, Card, ErrorState, Icon, LoadingState, formatJalaliDateTime, formatNumber, formatRelative,
  messageFor, useAsync, type AdminUserDetail,
} from "@vira/shared";
import { adminApi } from "../api";
import { useAuth } from "../auth";
import { StatusBadge, initials, statusLabel } from "./UsersPage";

const platformLabel: Record<string, string> = { android: "اندروید", ios: "iOS", web: "وب" };

/**
 * Side desk for one account (archive pattern: the moderation console's "inspection desk"). Every status change
 * needs a written reason (doc 08 §4.3) and is audited server-side; viewing the detail is itself audited.
 */
export function UserPanel({ id, onClose, onChanged }: { id: string; onClose: () => void; onChanged: () => void }) {
  const { can } = useAuth();
  const { data: u, error, loading, reload } = useAsync((signal) => adminApi.get<AdminUserDetail>(`/v1/admin/users/${id}`, { signal }).then((r) => r.data), [id]);
  const [next, setNext] = useState("");
  const [reason, setReason] = useState("");
  const [busy, setBusy] = useState<"status" | "revoke" | null>(null);
  const [msg, setMsg] = useState<{ tone: "success" | "danger"; text: string } | null>(null);
  const writable = can("users.write");

  async function changeStatus() {
    setBusy("status");
    setMsg(null);
    try {
      await adminApi.post(`/v1/admin/users/${id}/status`, { status: next, reason: reason.trim() });
      setMsg({ tone: "success", text: `وضعیت حساب به «${statusLabel[next]?.label ?? next}» تغییر کرد.` });
      setNext("");
      setReason("");
      reload();
      onChanged();
    } catch (e) {
      setMsg({ tone: "danger", text: e instanceof ApiError ? e.title : messageFor("internal_error") });
    } finally {
      setBusy(null);
    }
  }

  async function revoke() {
    setBusy("revoke");
    setMsg(null);
    try {
      const r = await adminApi.post<{ revoked: number }>(`/v1/admin/users/${id}/sessions/revoke`);
      setMsg({ tone: "success", text: `${formatNumber(r.data.revoked)} نشست فعال باطل شد.` });
      reload();
      onChanged();
    } catch (e) {
      setMsg({ tone: "danger", text: e instanceof ApiError ? e.title : messageFor("internal_error") });
    } finally {
      setBusy(null);
    }
  }

  return (
    <aside className="v-desk__panel" aria-label="میز بررسی حساب کاربری">
      <Card title={<span style={{ display: "inline-flex", gap: 8, alignItems: "center" }}><Icon name="person" />میز بررسی حساب</span>}
        actions={<Button variant="ghost" icon="close" aria-label="بستن" title="بستن" onClick={onClose} />}>
        {loading && !u ? <LoadingState /> : error ? <ErrorState error={error} onRetry={reload} /> : u && (
          <div style={{ display: "grid", gap: 16 }}>
            <div className="v-person">
              <span className="v-person__avatar" aria-hidden>{initials(u.display_name ?? "", u.handle)}</span>
              <div>
                <div className="v-person__name">{u.display_name || u.handle}</div>
                <div className="v-person__sub"><span className="ltr">@{u.handle}</span></div>
              </div>
              <span style={{ marginInlineStart: "auto" }}><StatusBadge status={u.status} /></span>
            </div>
            {u.status_reason && <div className="v-banner v-banner--warning">دلیل وضعیت فعلی: {u.status_reason}</div>}
            <dl className="v-dl">
              <dt>شناسه</dt><dd><span className="ltr" style={{ font: "var(--type-caption)" }}>{u.id}</span></dd>
              <dt>شماره همراه</dt><dd><span className="ltr tabular">{u.phone_masked || "—"}</span></dd>
              <dt>حساب</dt><dd>{u.is_private ? <Badge tone="neutral" icon="lock">خصوصی</Badge> : <Badge tone="info">عمومی</Badge>}</dd>
              <dt>پست / دنبال‌کننده / دنبال‌شونده</dt>
              <dd className="tabular">{formatNumber(u.posts ?? 0)} / {formatNumber(u.followers ?? 0)} / {formatNumber(u.following ?? 0)}</dd>
              <dt>گزارش‌های باز</dt><dd className="tabular">{formatNumber(u.open_reports ?? 0)}</dd>
              <dt>قصد فروش</dt><dd>{u.intends_to_sell ? "بله (فروشندگی از فاز ۴)" : "خیر"}</dd>
              <dt>عضویت</dt><dd>{formatJalaliDateTime(u.created_at)}</dd>
              {u.bio && <><dt>بیو</dt><dd>{u.bio}</dd></>}
            </dl>

            <section className="v-desk__section">
              <h3><Icon name="devices" />نشست‌ها و دستگاه‌ها</h3>
              {(u.sessions ?? []).length === 0 ? <p className="v-person__sub">نشستی ثبت نشده است.</p> : (
                <ul className="v-list">
                  {(u.sessions ?? []).map((x) => (
                    <li key={x.id}>
                      <span>{platformLabel[x.platform ?? ""] ?? x.platform}{x.model ? ` · ${x.model}` : ""}</span>
                      <span>{x.revoked_at ? <Badge tone="neutral">باطل‌شده</Badge> : <>فعال · {x.last_used_at ? formatRelative(x.last_used_at) : ""}</>}</span>
                    </li>
                  ))}
                </ul>
              )}
              {writable && (
                <Button variant="secondary" icon="logout" loading={busy === "revoke"} disabled={(u.active_sessions ?? 0) === 0}
                  onClick={() => void revoke()}>خروج از همهٔ دستگاه‌ها</Button>
              )}
            </section>

            <section className="v-desk__section">
              <h3><Icon name="history" />تاریخچهٔ وضعیت</h3>
              {(u.status_history ?? []).length === 0 ? <p className="v-person__sub">تغییری ثبت نشده است.</p> : (
                <ul className="v-list">
                  {(u.status_history ?? []).map((h, i) => (
                    <li key={i}>
                      <span>{statusLabel[h.from ?? ""]?.label ?? h.from} ← {statusLabel[h.to ?? ""]?.label ?? h.to}{h.reason ? ` — ${h.reason}` : ""}</span>
                      <span>{h.created_at ? formatJalaliDateTime(h.created_at) : ""}</span>
                    </li>
                  ))}
                </ul>
              )}
            </section>

            {writable ? (
              <section className="v-desk__section">
                <h3><Icon name="gavel" />اقدام و صدور حکم</h3>
                <form style={{ display: "grid", gap: 8 }} onSubmit={(e) => { e.preventDefault(); void changeStatus(); }}>
                  <div className="v-field">
                    <label htmlFor="user-next-status">وضعیت جدید</label>
                    <select id="user-next-status" className="v-select" value={next} onChange={(e) => setNext(e.target.value)} required>
                      <option value="">انتخاب کنید…</option>
                      {["active", "restricted", "suspended"].filter((k) => k !== u.status).map((k) => <option key={k} value={k}>{statusLabel[k].label}</option>)}
                    </select>
                  </div>
                  <div className="v-field">
                    <label htmlFor="user-status-reason">دلیل (الزامی، در گزارش ممیزی ثبت می‌شود)</label>
                    <textarea id="user-status-reason" className="v-textarea" value={reason} onChange={(e) => setReason(e.target.value)} maxLength={500} required />
                  </div>
                  {next === "suspended" && <div className="v-banner v-banner--warning">مسدودسازی همهٔ نشست‌های کاربر را باطل می‌کند.</div>}
                  <Button type="submit" variant={next === "suspended" ? "danger" : "primary"} loading={busy === "status"}
                    disabled={!next || !reason.trim()}>ثبت تغییر وضعیت</Button>
                </form>
              </section>
            ) : <div className="v-banner v-banner--info">نقش شما فقط اجازهٔ مشاهده دارد.</div>}
            {msg && <div role="status" className={`v-banner ${msg.tone === "success" ? "v-banner--info" : "v-banner--warning"}`}>{msg.text}</div>}
          </div>
        )}
      </Card>
    </aside>
  );
}
