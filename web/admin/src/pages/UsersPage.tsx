import { useState, type ReactNode } from "react";
import {
  ApiError, Badge, Button, Card, EmptyState, ErrorState, Icon, LoadingState, PageHeader, formatJalaliDate, formatNumber,
  formatRelative, toPersianDigits, useAsync, type AdminUser, type AdminUsersSummary, type IconName, type Tone,
} from "@vira/shared";
import { adminApi } from "../api";
import { UserPanel } from "./UserPanel";

// Reference: super-admin/vira_users_management (archive). Layout kept: breadcrumb + title, four KPI tiles, filter
// panel with active-filter chips, user table, pager. Corrections (doc 19): phone numbers are masked server-side
// and never searchable by partial number; e-mail is not collected in P2; bulk-select checkboxes and the CSV export
// have no P2 backend and are omitted rather than shown as fake controls (see docs/progress/P2.md).

interface Page { data: AdminUser[]; next_cursor: string | null }

export const statusLabel: Record<string, { label: string; tone: Tone; icon: IconName }> = {
  active: { label: "فعال", tone: "success", icon: "check_circle" },
  restricted: { label: "محدودشده", tone: "warning", icon: "warning" },
  suspended: { label: "مسدود", tone: "danger", icon: "block" },
  deletion_pending: { label: "در انتظار حذف", tone: "neutral", icon: "history" },
  deleted: { label: "حذف‌شده", tone: "neutral", icon: "person_off" },
};

export function StatusBadge({ status }: { status: string }) {
  const s = statusLabel[status] ?? { label: status, tone: "neutral" as Tone, icon: "person" as IconName };
  return <Badge tone={s.tone} icon={s.icon}>{s.label}</Badge>;
}

/** Two-letter avatar from the display name (archive: coloured initials tile). */
export function initials(name: string, handle: string): string {
  const [a = "", b = ""] = (name || handle).trim().split(/\s+/).filter(Boolean);
  if (!a) return "؟";
  return b ? a.slice(0, 1) + b.slice(0, 1) : a.slice(0, 2);
}

export function UsersPage() {
  const [q, setQ] = useState("");
  const [status, setStatus] = useState("");
  const [applied, setApplied] = useState({ q: "", status: "" });
  const [cursors, setCursors] = useState<string[]>([]); // cursors of the pages before the current one
  const [cursor, setCursor] = useState<string | undefined>();
  const [selected, setSelected] = useState<string | null>(null);

  const summary = useAsync((signal) => adminApi.get<AdminUsersSummary>("/v1/admin/users/summary", { signal }).then((r) => r.data), []);
  const list = useAsync(async (signal) => {
    const p = new URLSearchParams({ limit: "20" });
    if (applied.q) p.set("q", applied.q);
    if (applied.status) p.set("status", applied.status);
    if (cursor) p.set("cursor", cursor);
    return (await adminApi.get<Page>(`/v1/admin/users?${p}`, { signal })).data;
  }, [applied, cursor]);

  function apply(next: { q: string; status: string }) {
    setApplied(next);
    setCursors([]);
    setCursor(undefined);
  }
  function reloadAll() {
    summary.reload();
    list.reload();
  }

  const s = summary.data;
  const rows = list.data?.data ?? [];
  const page = cursors.length + 1;
  const badFilter = list.error instanceof ApiError && list.error.status === 422;

  return (
    <>
      <div>
        <nav className="v-breadcrumb" aria-label="مسیر">
          <span>سامانه جامع ویرا</span><Icon name="chevron_left" /><span>فاز دوم</span><Icon name="chevron_left" />
          <strong>مدیریت کاربران و سطوح دسترسی</strong>
        </nav>
        <PageHeader title="کنسول نظارت و راهبری کاربران" />
      </div>

      <div className="v-kpis">
        <Kpi label="کل کاربران فعال" icon="group" value={s?.active} loading={summary.loading}
          foot={<><span>نشست‌های فعال</span><strong className="tabular">{s ? formatNumber(s.active_sessions) : "—"}</strong></>} />
        <Kpi label="ثبت‌نام‌های امروز" icon="how_to_reg" tone="success" value={s?.signups_today} loading={summary.loading}
          foot={<><span>کل حساب‌ها</span><strong className="tabular">{s ? formatNumber(s.total) : "—"}</strong></>} />
        <Kpi label="نرخ احراز هویت شاهکار" icon="verified_user" tone="info" locked="فاز ۴"
          foot={<span>استعلام شاهکار با احراز هویت فروشنده (فاز ۴) فعال می‌شود</span>} />
        <Kpi label="حساب‌های مسدود یا محدود" icon="gavel" tone="danger" value={s ? s.suspended + s.restricted : undefined} loading={summary.loading}
          badge={s && s.suspended + s.restricted > 0 ? <Badge tone="danger">اقدام لازم</Badge> : undefined}
          foot={<><span>مسدود / محدود</span><strong className="tabular">{s ? `${formatNumber(s.suspended)} / ${formatNumber(s.restricted)}` : "—"}</strong></>} />
      </div>
      {summary.error ? <Card><ErrorState error={summary.error} onRetry={summary.reload} /></Card> : null}

      <Card>
        <form className="v-filters" onSubmit={(e) => { e.preventDefault(); apply({ q: q.trim(), status }); }}>
          <div className="v-filters__row">
            <div className="v-field">
              <label htmlFor="users-q">نام کاربری، شناسه یا شماره همراه</label>
              <input id="users-q" className="v-input" value={q} onChange={(e) => setQ(e.target.value)}
                placeholder="مثال: @sara یا ۰۹۱۲۱۲۳۴۵۶۷" aria-invalid={badFilter || undefined} />
            </div>
            <div className="v-field">
              <label htmlFor="users-role">نقش</label>
              <select id="users-role" className="v-select" disabled title="نقش فروشنده از فاز ۴ فعال می‌شود">
                <option>همه نقش‌ها (کاربر)</option>
              </select>
            </div>
            <div className="v-field">
              <label htmlFor="users-status">وضعیت حساب</label>
              <select id="users-status" className="v-select" value={status} onChange={(e) => { setStatus(e.target.value); apply({ q: q.trim(), status: e.target.value }); }}>
                <option value="">همه وضعیت‌ها</option>
                {Object.entries(statusLabel).map(([k, v]) => <option key={k} value={k}>{v.label}</option>)}
              </select>
            </div>
            <div style={{ display: "flex", gap: 8 }}>
              <Button type="submit" variant="secondary" icon="search">جستجو</Button>
              <Button type="button" variant="ghost" icon="filter_alt_off" aria-label="پاک کردن فیلترها" title="پاک کردن فیلترها"
                onClick={() => { setQ(""); setStatus(""); apply({ q: "", status: "" }); }} />
              <Button type="button" variant="ghost" icon="refresh" aria-label="بارگذاری دوباره" title="بارگذاری دوباره" onClick={reloadAll} />
            </div>
          </div>
          <div className="v-filters__meta">
            <div className="v-chips">
              <span>فیلترهای فعال:</span>
              {!applied.q && !applied.status && <span className="v-chip">همه کاربران</span>}
              {applied.q && <span className="v-chip">جستجو: <span className="ltr">{applied.q}</span>
                <button type="button" aria-label="حذف فیلتر جستجو" onClick={() => { setQ(""); apply({ ...applied, q: "" }); }}><Icon name="close" /></button></span>}
              {applied.status && <span className="v-chip">وضعیت: {statusLabel[applied.status]?.label}
                <button type="button" aria-label="حذف فیلتر وضعیت" onClick={() => { setStatus(""); apply({ ...applied, status: "" }); }}><Icon name="close" /></button></span>}
            </div>
            <span>{list.data ? `نمایش ${toPersianDigits(rows.length)} کاربر در صفحه ${toPersianDigits(page)}` : ""}</span>
          </div>
        </form>
      </Card>

      <div className={selected ? "v-desk" : undefined}>
        <div className="v-table-wrap">
          {list.loading && !list.data ? <div style={{ padding: 20 }}><LoadingState /></div>
            : list.error ? <ErrorState error={list.error} onRetry={list.reload} />
            : rows.length === 0 ? <EmptyState icon="group" title="کاربری یافت نشد" description="فیلترها را تغییر دهید. جستجو با شماره همراه فقط با شمارهٔ کامل انجام می‌شود." />
            : (
              <table className="v-table">
                <thead><tr>
                  <th>نام و شناسه</th><th>شماره تماس</th><th>نقش</th><th>تاریخ عضویت</th><th>آخرین ورود</th><th>وضعیت حساب</th><th>عملیات</th>
                </tr></thead>
                <tbody>
                  {rows.map((u) => (
                    <tr key={u.id} data-selected={selected === u.id || undefined} data-tone={u.status === "suspended" ? "danger" : undefined}>
                      <td>
                        <div className="v-person">
                          <span className="v-person__avatar" aria-hidden>{initials(u.display_name ?? "", u.handle)}</span>
                          <div>
                            <div className="v-person__name">{u.display_name || u.handle}</div>
                            <div className="v-person__sub"><span className="ltr">@{u.handle}</span> · UID: <span className="ltr">{u.id.slice(0, 8)}</span></div>
                          </div>
                        </div>
                      </td>
                      <td><span className="ltr tabular">{u.phone_masked || "—"}</span></td>
                      <td><Badge tone="info">کاربر</Badge></td>
                      <td className="tabular">{formatJalaliDate(u.created_at)}</td>
                      <td>{u.last_login_at ? formatRelative(u.last_login_at) : "—"}</td>
                      <td>
                        <StatusBadge status={u.status} />
                        {(u.open_reports ?? 0) > 0 && <div className="v-person__sub" style={{ marginBlockStart: 4 }}>{toPersianDigits(u.open_reports ?? 0)} گزارش باز</div>}
                      </td>
                      <td>
                        <Button variant="ghost" icon="visibility" aria-label={`مشاهدهٔ ${u.handle}`} title="مشاهده و مدیریت"
                          onClick={() => setSelected(u.id === selected ? null : u.id)} />
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
          <div className="v-pager">
            <span>صفحهٔ {toPersianDigits(page)} · {s ? `از کل ${formatNumber(s.total)} پرونده کاربری` : ""}</span>
            <div className="v-pager__btns">
              <Button variant="secondary" disabled={cursors.length === 0 || list.loading}
                onClick={() => { const prev = [...cursors]; const c = prev.pop(); setCursors(prev); setCursor(c || undefined); }}>قبلی</Button>
              <span className="v-pager__page" aria-current="page">{toPersianDigits(page)}</span>
              <Button variant="secondary" disabled={!list.data?.next_cursor || list.loading}
                onClick={() => { setCursors([...cursors, cursor ?? ""]); setCursor(list.data?.next_cursor ?? undefined); }}>بعدی</Button>
            </div>
          </div>
        </div>
        {selected && <UserPanel id={selected} onClose={() => setSelected(null)} onChanged={reloadAll} />}
      </div>
    </>
  );
}

function Kpi({ label, icon, tone, value, loading, foot, badge, locked }: {
  label: string; icon: IconName; tone?: "success" | "danger" | "info"; value?: number; loading?: boolean;
  foot: ReactNode; badge?: ReactNode; locked?: string;
}) {
  return (
    <Card>
      <div className="v-kpi">
        <div className="v-kpi__head">
          <span className="v-kpi__label">{label}</span>
          <span className={`v-kpi__icon ${tone ? `v-kpi__icon--${tone}` : ""}`}><Icon name={icon} /></span>
        </div>
        <div className="v-kpi__row">
          <span className="v-kpi__value">{locked ? "—" : loading && value === undefined ? "…" : value === undefined ? "—" : formatNumber(value)}</span>
          {locked ? <Badge tone="neutral" icon="lock">{locked}</Badge> : badge}
        </div>
        <div className="v-kpi__foot">{foot}</div>
      </div>
    </Card>
  );
}

