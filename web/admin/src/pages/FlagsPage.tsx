import { useState } from "react";
import { ApiError, Badge, Button, Card, EmptyState, ErrorState, LoadingState, PageHeader, Switch, TextField, formatRelative, useAsync, type Flag } from "@vira/shared";
import { adminApi } from "../api";
import { useAuth } from "../auth";

/** Feature flags (doc 06 §8). Regulated flags are shown locked with their gate; the switch is disabled and the
 * server refuses changes anyway (409 feature_locked). Non-regulated changes need a reason and the current
 * version (If-Match), and are audited. */
export function FlagsPage() {
  const { can } = useAuth();
  const { data, error, loading, reload } = useAsync((signal) => adminApi.get<{ data: Flag[] }>("/v1/admin/flags", { signal }).then((r) => r.data.data), []);
  const [pending, setPending] = useState<string>();
  const [message, setMessage] = useState<{ tone: "success" | "danger"; text: string }>();
  const [editing, setEditing] = useState<{ flag: Flag; enabled: boolean }>();
  const [reason, setReason] = useState("");

  async function save() {
    if (!editing || !reason.trim()) return;
    const { flag: f, enabled } = editing;
    setPending(f.key);
    setMessage(undefined);
    try {
      await adminApi.put(`/v1/admin/flags/${encodeURIComponent(f.key)}`, { enabled, reason: reason.trim() }, { headers: { "If-Match": `"${f.version}"` } });
      setMessage({ tone: "success", text: "تغییر ذخیره شد." });
      setEditing(undefined);
      setReason("");
      reload();
    } catch (e) {
      setMessage({ tone: "danger", text: e instanceof ApiError ? e.title : "خطا در ذخیره" });
    } finally {
      setPending(undefined);
    }
  }

  const writable = can("flags.write");
  return (
    <>
      <PageHeader title="فیچر فلگ‌ها و پیکربندی" subtitle="فلگ‌های قانون‌مند تا عبور از دروازهٔ تأیید قفل‌اند." />
      {editing && (
        <Card title={`${editing.enabled ? "روشن" : "خاموش"} کردن ${editing.flag.key}`}>
          <form style={{ display: "flex", gap: 12, alignItems: "flex-end", flexWrap: "wrap" }} onSubmit={(e) => { e.preventDefault(); void save(); }}>
            <div style={{ flex: 1, minInlineSize: 280 }}>
              <TextField label="دلیل تغییر (در گزارش ممیزی ثبت می‌شود)" value={reason} autoFocus onChange={(e) => setReason(e.target.value)} />
            </div>
            <Button type="submit" loading={pending === editing.flag.key} disabled={!reason.trim()}>ذخیره</Button>
            <Button type="button" variant="ghost" onClick={() => { setEditing(undefined); setReason(""); }}>انصراف</Button>
          </form>
        </Card>
      )}
      {message && <div className={`v-banner v-banner--${message.tone === "success" ? "info" : "warning"}`} role="status">{message.text}</div>}
      <Card>
        {loading && !data ? <LoadingState /> : error ? <ErrorState error={error} onRetry={reload} /> : !data?.length ? (
          <EmptyState icon="flag" title="فلگی تعریف نشده است" />
        ) : (
          <table className="v-table">
            <thead><tr><th>کلید</th><th>شرح</th><th>مخاطب</th><th>وضعیت</th><th>آخرین تغییر</th><th>فعال</th></tr></thead>
            <tbody>
              {data.map((f) => (
                <tr key={f.key} data-testid={`flag-${f.key}`}>
                  <td><span className="ltr">{f.key}</span></td>
                  <td>{f.description}</td>
                  <td>{f.audiences.map((a) => <Badge key={a}>{a}</Badge>)}</td>
                  <td>{f.regulated ? <Badge tone="danger" icon="lock">قفل تا {f.gate}</Badge> : <Badge tone={f.enabled ? "success" : "neutral"}>{f.enabled ? "روشن" : "خاموش"}</Badge>}</td>
                  <td>{f.updated_at ? formatRelative(f.updated_at) : "—"}</td>
                  <td>
                    <Switch label={`فعال‌سازی ${f.key}`} checked={f.enabled} disabled={f.regulated || !writable || pending === f.key}
                      onChange={(v) => { setEditing({ flag: f, enabled: v }); setReason(""); }} />
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </Card>
    </>
  );
}
