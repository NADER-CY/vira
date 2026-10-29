import { useState } from "react";
import { Badge, Button, Card, EmptyState, ErrorState, LoadingState, PageHeader, TextField, formatJalaliDateTime, useAsync, type AuditEvent } from "@vira/shared";
import { adminApi } from "../api";

interface Page { data: AuditEvent[]; next_cursor: string | null }

const outcomeTone = { success: "success", denied: "danger", failure: "warning" } as const;

/** Append-only audit log (doc 08). Read-only; filter by action; cursor pagination. */
export function AuditPage() {
  const [action, setAction] = useState("");
  const [applied, setApplied] = useState("");
  const [extra, setExtra] = useState<AuditEvent[]>([]);
  const [cursor, setCursor] = useState<string | null>(null);
  const [moreLoading, setMoreLoading] = useState(false);

  const q = (c?: string) => {
    const p = new URLSearchParams({ limit: "50" });
    if (applied) p.set("action", applied);
    if (c) p.set("cursor", c);
    return `/v1/admin/audit-events?${p}`;
  };
  const { data, error, loading, reload } = useAsync(async (signal) => {
    const r = await adminApi.get<Page>(q(), { signal });
    setExtra([]);
    setCursor(r.data.next_cursor);
    return r.data.data;
  }, [applied]);

  async function more() {
    if (!cursor) return;
    setMoreLoading(true);
    try {
      const r = await adminApi.get<Page>(q(cursor));
      setExtra((x) => [...x, ...r.data.data]);
      setCursor(r.data.next_cursor);
    } finally {
      setMoreLoading(false);
    }
  }

  const rows = [...(data ?? []), ...extra];
  return (
    <>
      <PageHeader title="گزارش ممیزی" subtitle="ثبت تغییرناپذیر ورودها و اقدام‌های مدیریتی" />
      <Card>
        <form style={{ display: "flex", gap: 12, alignItems: "flex-end", marginBlockEnd: 16 }} onSubmit={(e) => { e.preventDefault(); setApplied(action.trim()); }}>
          <div style={{ flex: 1, maxInlineSize: 360 }}>
            <TextField label="فیلتر اقدام (مثلاً admin.login یا flags.update)" dir="ltr" value={action} onChange={(e) => setAction(e.target.value)} />
          </div>
          <Button type="submit" variant="secondary" icon="search">اعمال</Button>
        </form>
        {loading && !data ? <LoadingState /> : error ? <ErrorState error={error} onRetry={reload} /> : rows.length === 0 ? (
          <EmptyState icon="history" title="رویدادی یافت نشد" />
        ) : (
          <>
            <table className="v-table">
              <thead><tr><th>زمان</th><th>اقدام</th><th>کنشگر</th><th>هدف</th><th>نتیجه</th><th>IP</th></tr></thead>
              <tbody>
                {rows.map((e) => (
                  <tr key={e.id}>
                    <td>{formatJalaliDateTime(e.created_at)}</td>
                    <td><span className="ltr">{e.action}</span></td>
                    <td>{e.actor_type}</td>
                    <td><span className="ltr">{[e.target_type, e.target_id].filter(Boolean).join(":") || "—"}</span></td>
                    <td><Badge tone={outcomeTone[e.outcome as keyof typeof outcomeTone] ?? "neutral"}>{e.outcome}</Badge></td>
                    <td><span className="ltr">{e.ip ?? "—"}</span></td>
                  </tr>
                ))}
              </tbody>
            </table>
            {cursor && <div style={{ marginBlockStart: 16 }}><Button variant="secondary" loading={moreLoading} onClick={() => void more()}>موارد بیشتر</Button></div>}
          </>
        )}
      </Card>
    </>
  );
}
