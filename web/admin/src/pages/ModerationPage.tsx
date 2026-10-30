import { useState, type ReactNode } from "react";
import {
  ApiError, Badge, Button, Card, EmptyState, ErrorState, Icon, LoadingState, PageHeader, formatJalaliDateTime, formatNumber,
  formatRelative, messageFor, toPersianDigits, useAsync, type AdminReport, type AdminReportsSummary, type IconName, type Tone,
} from "@vira/shared";
import { adminApi } from "../api";
import { useAuth } from "../auth";

// Reference: super-admin/vira_comments_reports_dark (archive). Layout kept: breadcrumb + title, KPI tiles, queue
// tabs, live queue table on the start side and the "inspection & resolution desk" on the end side with the final
// verdict buttons and a confidential note. Tabs 2–3 (AI comment audit, anti-spam) belong to later phases and are
// shown disabled; the "instant settlement" header button of the archive is not reproduced (doc 19, V-06).

interface Page { data: AdminReport[]; next_cursor: string | null }

const reasonLabel: Record<string, string> = {
  spam: "هرزنامه", harassment: "آزار و فحاشی", hate: "نفرت‌پراکنی", violence: "خشونت", nudity: "محتوای جنسی",
  scam: "کلاهبرداری", counterfeit: "کالای تقلبی", misinformation: "اطلاعات نادرست", self_harm: "آسیب به خود", other: "سایر",
};
const targetLabel: Record<string, { label: string; icon: IconName }> = {
  post: { label: "پست", icon: "video_library" }, comment: { label: "نظر", icon: "chat" }, user: { label: "حساب کاربری", icon: "person" },
};
const stateLabel: Record<string, { label: string; tone: Tone }> = {
  open: { label: "در انتظار بررسی", tone: "warning" }, actioned: { label: "اقدام‌شده", tone: "success" }, dismissed: { label: "رد گزارش", tone: "neutral" },
};
const resolutions: { key: string; label: string; icon: IconName; variant: "danger" | "secondary"; for: string[] }[] = [
  { key: "remove_content", label: "تأیید گزارش و حذف محتوا", icon: "block", variant: "danger", for: ["post", "comment"] },
  { key: "restrict_user", label: "محدودسازی حساب مالک", icon: "warning", variant: "secondary", for: ["post", "comment", "user"] },
  { key: "suspend_user", label: "تعلیق حساب مالک", icon: "person_off", variant: "danger", for: ["post", "comment", "user"] },
  { key: "dismiss", label: "رد گزارش (بدون تخلف)", icon: "check_circle", variant: "secondary", for: ["post", "comment", "user"] },
];

export function ModerationPage() {
  const { can } = useAuth();
  const [state, setState] = useState("open");
  const [type, setType] = useState("");
  const [cursors, setCursors] = useState<string[]>([]);
  const [cursor, setCursor] = useState<string | undefined>();
  const [selected, setSelected] = useState<AdminReport | null>(null);

  const summary = useAsync((signal) => adminApi.get<AdminReportsSummary>("/v1/admin/reports/summary", { signal }).then((r) => r.data), []);
  const list = useAsync(async (signal) => {
    const p = new URLSearchParams({ limit: "20" });
    if (state) p.set("state", state);
    if (type) p.set("target_type", type);
    if (cursor) p.set("cursor", cursor);
    return (await adminApi.get<Page>(`/v1/admin/reports?${p}`, { signal })).data;
  }, [state, type, cursor]);

  function resetPaging() { setCursors([]); setCursor(undefined); }
  const s = summary.data;
  const rows = list.data?.data ?? [];
  const page = cursors.length + 1;

  return (
    <>
      <div>
        <nav className="v-breadcrumb" aria-label="مسیر">
          <span>نظارت و امنیت پلتفرم</span><Icon name="chevron_left" /><strong>مرکز رسیدگی به گزارش‌های تخلف</strong>
        </nav>
        <PageHeader title="مرکز ممیزی نظرات، تعاملات و رسیدگی به تخلفات"
          actions={<Button variant="secondary" icon="refresh" onClick={() => { summary.reload(); list.reload(); }}>بروزرسانی</Button>} />
      </div>

      <div className="v-kpis">
        <Kpi label="کل گزارش‌های ثبت‌شده" icon="flag" value={s?.total} sub={s ? <>{formatNumber(s.open)} پرونده باز</> : null} />
        <Kpi label="صف بررسی دستی" icon="report" tone="danger" value={s?.open}
          sub={s ? <>پست {formatNumber(s.open_posts)} · نظر {formatNumber(s.open_comments)} · حساب {formatNumber(s.open_users)}</> : null} />
        <Kpi label="تعاملات مشکوک و اسپم" icon="smart_toy" tone="info" locked="فاز ۹" sub={<>تشخیص خودکار در فاز هوش مصنوعی</>} />
        <Kpi label="حل‌وفصل‌شده امروز" icon="check_circle" tone="success" value={s ? s.actioned_today + s.dismissed_today : undefined}
          sub={s ? <>اقدام {formatNumber(s.actioned_today)} · رد {formatNumber(s.dismissed_today)}</> : null} />
      </div>
      {summary.error ? <Card><ErrorState error={summary.error} onRetry={summary.reload} /></Card> : null}

      <div className="v-tabs" role="tablist" aria-label="صف‌های بررسی">
        <button type="button" role="tab" className="v-tab" aria-selected="true">
          <Icon name="warning" />۱. صف گزارش‌های کاربران و تخلفات {s && <Badge tone="danger">{formatNumber(s.open)}</Badge>}
        </button>
        <button type="button" role="tab" className="v-tab" aria-selected="false" disabled title="فاز ۹">
          <Icon name="chat" />۲. ممیزی نظرات و بازخوردهای ارسالی <Badge tone="neutral">فاز ۹</Badge>
        </button>
        <button type="button" role="tab" className="v-tab" aria-selected="false" disabled title="فاز ۹">
          <Icon name="shield_lock" />۳. پایش تعاملات مشکوک، لایک/فالو فیک و اسپم <Badge tone="neutral">فاز ۹</Badge>
        </button>
      </div>

      <div className="v-desk">
        <div className="v-table-wrap">
          <div className="v-pager" style={{ borderBlockEnd: "1px solid var(--color-outline)" }}>
            <strong style={{ color: "var(--color-on-surface)", font: "var(--type-title-sm, var(--type-label-md))" }}>صف زنده بررسی تخلفات</strong>
            <div className="v-pager__btns">
              <select className="v-select" aria-label="وضعیت" value={state} onChange={(e) => { setState(e.target.value); setSelected(null); resetPaging(); }}>
                <option value="open">در انتظار بررسی</option>
                <option value="actioned">اقدام‌شده</option>
                <option value="dismissed">ردشده</option>
                <option value="">همه</option>
              </select>
              <select className="v-select" aria-label="نوع مورد گزارش" value={type} onChange={(e) => { setType(e.target.value); resetPaging(); }}>
                <option value="">همه انواع</option>
                <option value="post">پست</option>
                <option value="comment">نظر</option>
                <option value="user">حساب کاربری</option>
              </select>
              <Button variant="ghost" icon="filter_alt_off" aria-label="پاک کردن فیلترها" title="پاک کردن فیلترها"
                onClick={() => { setState("open"); setType(""); resetPaging(); }} />
            </div>
          </div>
          {list.loading && !list.data ? <div style={{ padding: 20 }}><LoadingState /></div>
            : list.error ? <ErrorState error={list.error} onRetry={list.reload} />
            : rows.length === 0 ? <EmptyState icon="check_circle" title="صف خالی است" description="گزارشی با این فیلترها وجود ندارد." />
            : (
              <table className="v-table">
                <thead><tr><th>شناسه پرونده</th><th>گزارش‌دهنده</th><th>مورد گزارش / نوع</th><th>علت تخلف</th><th>ثبت</th><th>وضعیت</th><th>عملیات</th></tr></thead>
                <tbody>
                  {rows.map((r) => (
                    <tr key={r.id} data-selected={selected?.id === r.id || undefined}>
                      <td className="nowrap"><span className="ltr tabular">#{r.id.slice(0, 8)}</span></td>
                      <td>{r.reporter_handle ? <span className="ltr">@{r.reporter_handle}</span> : "—"}</td>
                      <td>
                        <div className="v-person">
                          <Icon name={targetLabel[r.target_type]?.icon ?? "flag"} />
                          <div>
                            <div className="v-person__name">{targetLabel[r.target_type]?.label ?? r.target_type}
                              {r.target_owner_handle && <> · <span className="ltr">@{r.target_owner_handle}</span></>}</div>
                            {r.target_preview && <div className="v-person__sub" style={{ maxInlineSize: 200, overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}>{r.target_preview}</div>}
                          </div>
                        </div>
                      </td>
                      <td><Badge tone="danger">{reasonLabel[r.reason] ?? r.reason}</Badge>
                        {(r.open_reports_on_target ?? 0) > 1 && <div className="v-person__sub">{toPersianDigits(r.open_reports_on_target ?? 0)} گزارش روی این مورد</div>}</td>
                      <td className="nowrap">{formatRelative(r.created_at)}</td>
                      <td><Badge tone={stateLabel[r.state]?.tone ?? "neutral"}>{stateLabel[r.state]?.label ?? r.state}</Badge></td>
                      <td><Button variant={r.state === "open" ? "secondary" : "ghost"} onClick={() => setSelected(r)}>{r.state === "open" ? "بررسی" : "مشاهده"}</Button></td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
          <div className="v-pager">
            <span>صفحهٔ {toPersianDigits(page)}{s ? ` · ${formatNumber(s.open)} پرونده فعال` : ""}</span>
            <div className="v-pager__btns">
              <Button variant="secondary" disabled={cursors.length === 0 || list.loading}
                onClick={() => { const prev = [...cursors]; const c = prev.pop(); setCursors(prev); setCursor(c || undefined); }}>قبلی</Button>
              <span className="v-pager__page" aria-current="page">{toPersianDigits(page)}</span>
              <Button variant="secondary" disabled={!list.data?.next_cursor || list.loading}
                onClick={() => { setCursors([...cursors, cursor ?? ""]); setCursor(list.data?.next_cursor ?? undefined); }}>بعدی</Button>
            </div>
          </div>
        </div>

        <ReportDesk key={selected?.id ?? "none"} report={selected} writable={can("reports.write")}
          onResolved={(r) => { setSelected(r); summary.reload(); list.reload(); }} />
      </div>
    </>
  );
}

function ReportDesk({ report: r, writable, onResolved }: { report: AdminReport | null; writable: boolean; onResolved: (r: AdminReport) => void }) {
  const [note, setNote] = useState("");
  const [busy, setBusy] = useState<string | null>(null);
  const [err, setErr] = useState<string | null>(null);

  async function resolve(resolution: string) {
    if (!r) return;
    setBusy(resolution);
    setErr(null);
    try {
      const res = await adminApi.post<AdminReport>(`/v1/admin/reports/${r.id}/resolve`, { resolution, note: note.trim() });
      setNote("");
      onResolved({ ...r, ...res.data });
    } catch (e) {
      setErr(e instanceof ApiError ? e.title : messageFor("internal_error"));
    } finally {
      setBusy(null);
    }
  }

  return (
    <aside className="v-desk__panel" aria-label="میز بازرسی و حل اختلاف پرونده">
      <Card title={<span style={{ display: "inline-flex", gap: 8, alignItems: "center" }}><Icon name="gavel" />میز بازرسی و حل اختلاف پرونده</span>}>
        {!r ? <EmptyState icon="report" title="پرونده‌ای انتخاب نشده" description="یک گزارش را از صف برای بررسی انتخاب کنید." /> : (
          <div style={{ display: "grid", gap: 16 }}>
            <div className="v-person__sub">شناسه پرونده: <span className="ltr">{r.id}</span></div>
            <dl className="v-dl">
              <dt>شاکی (گزارش‌دهنده)</dt><dd>{r.reporter_handle ? <span className="ltr">@{r.reporter_handle}</span> : "—"}</dd>
              <dt>مورد گزارش</dt><dd>{targetLabel[r.target_type]?.label ?? r.target_type}{r.target_owner_handle && <> · <span className="ltr">@{r.target_owner_handle}</span></>}</dd>
              <dt>علت</dt><dd><Badge tone="danger">{reasonLabel[r.reason] ?? r.reason}</Badge></dd>
              <dt>ثبت</dt><dd>{formatJalaliDateTime(r.created_at)}</dd>
              <dt>وضعیت</dt><dd><Badge tone={stateLabel[r.state]?.tone ?? "neutral"}>{stateLabel[r.state]?.label ?? r.state}</Badge></dd>
            </dl>
            {r.target_preview && (
              <section className="v-desk__section">
                <h3><Icon name="visibility" />محتوای گزارش‌شده</h3>
                <p className="v-banner v-banner--info">{r.target_preview}</p>
              </section>
            )}
            {r.details && (
              <section className="v-desk__section">
                <h3><Icon name="chat" />توضیح شاکی</h3>
                <p>{r.details}</p>
              </section>
            )}
            {r.state !== "open" ? (
              <div className="v-banner v-banner--info">
                این پرونده بسته شده است{r.resolution ? ` (${resolutions.find((x) => x.key === r.resolution)?.label ?? r.resolution})` : ""}.
                {r.resolution_note && <> یادداشت: {r.resolution_note}</>}
              </div>
            ) : writable ? (
              <section className="v-desk__section">
                <h3><Icon name="gavel" />اقدام نهایی ممیزی و صدور حکم</h3>
                <div className="v-field">
                  <label htmlFor="report-note">یادداشت محرمانه ممیز (الزامی)</label>
                  <textarea id="report-note" className="v-textarea" value={note} onChange={(e) => setNote(e.target.value)} maxLength={500} />
                </div>
                <div className="v-actions">
                  {resolutions.filter((x) => x.for.includes(r.target_type)).map((x) => (
                    <Button key={x.key} variant={x.variant} icon={x.icon} loading={busy === x.key} disabled={!note.trim() || (busy !== null && busy !== x.key)}
                      onClick={() => void resolve(x.key)}>{x.label}</Button>
                  ))}
                </div>
                <p className="v-person__sub">حکم روی همهٔ گزارش‌های باز همین مورد اعمال و در گزارش ممیزی ثبت می‌شود.</p>
              </section>
            ) : <div className="v-banner v-banner--info">نقش شما فقط اجازهٔ مشاهده دارد.</div>}
            {err && <div role="alert" className="v-banner v-banner--warning">{err}</div>}
          </div>
        )}
      </Card>
    </aside>
  );
}

function Kpi({ label, icon, tone, value, sub, locked }: { label: string; icon: IconName; tone?: "success" | "danger" | "info"; value?: number; sub: ReactNode; locked?: string }) {
  return (
    <Card>
      <div className="v-kpi">
        <div className="v-kpi__head">
          <span className="v-kpi__label">{label}</span>
          <span className={`v-kpi__icon ${tone ? `v-kpi__icon--${tone}` : ""}`}><Icon name={icon} /></span>
        </div>
        <div className="v-kpi__row">
          <span className="v-kpi__value">{locked || value === undefined ? "—" : formatNumber(value)}</span>
          {locked && <Badge tone="neutral" icon="lock">{locked}</Badge>}
        </div>
        <div className="v-kpi__foot"><span>{sub}</span></div>
      </div>
    </Card>
  );
}
