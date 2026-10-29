import { Badge, Card, ErrorState, LoadingState, PageHeader, formatJalaliDateTime, isolate, useAsync, type SystemStatus } from "@vira/shared";
import { adminApi } from "../api";

const gateLabel: Record<string, string> = {
  "payments.real_money": "پرداخت با پول واقعی",
  "settlement.live": "تسویهٔ زندهٔ فروشندگان",
  "payouts.withdrawals": "برداشت فروشندگان",
  "wallet.topup": "شارژ کیف پول",
  "usdt.deposits": "واریز/پرداخت USDT",
  "usdt.withdrawals": "برداشت USDT",
  "usdt.conversion": "تبدیل USDT ↔ تومان",
  "credit.bnpl": "اعتبار / خرید اقساطی",
};

export function DashboardPage() {
  const { data, error, loading, reload } = useAsync((signal) => adminApi.get<SystemStatus>("/v1/admin/system/status", { signal }).then((r) => r.data), []);
  return (
    <>
      <PageHeader title="داشبورد مدیریت ارشد" subtitle="وضعیت سامانه و قابلیت‌های قفل‌شده — فاز ۱ (زیرساخت)" />
      {loading && !data ? <Card><LoadingState /></Card> : error ? <Card><ErrorState error={error} onRetry={reload} /></Card> : data && (
        <>
          <div className="v-grid">
            <Card>
              <div className="v-stat__label">وضعیت سرویس مدیریت</div>
              <div className="v-stat__value">
                <Badge tone={data.readiness.status === "ok" ? "success" : "danger"}>{data.readiness.status === "ok" ? "پایدار" : "اختلال"}</Badge>
              </div>
            </Card>
            <Card>
              <div className="v-stat__label">محیط</div>
              <div className="v-stat__value" style={{ font: "var(--type-title-lg)" }}>{isolate(data.environment ?? "—")}</div>
            </Card>
            <Card>
              <div className="v-stat__label">نسخه</div>
              <div className="v-stat__value" style={{ font: "var(--type-title-lg)" }}>{isolate(data.version ?? "—")}</div>
            </Card>
            <Card>
              <div className="v-stat__label">زمان سرور</div>
              <div className="v-stat__value" style={{ font: "var(--type-title-md)" }}>{data.time ? formatJalaliDateTime(data.time) : "—"}</div>
            </Card>
          </div>
          <div className="v-grid" style={{ gridTemplateColumns: "repeat(auto-fit, minmax(360px, 1fr))" }}>
            <Card title="بررسی وابستگی‌ها">
              <table className="v-table">
                <thead><tr><th>وابستگی</th><th>وضعیت</th><th>زمان پاسخ</th></tr></thead>
                <tbody>
                  {data.readiness.checks.map((c) => (
                    <tr key={c.name}>
                      <td><span className="ltr">{c.name}</span></td>
                      <td><Badge tone={c.status === "ok" ? "success" : "danger"}>{c.status === "ok" ? "سالم" : "خطا"}</Badge></td>
                      <td className="tabular">{c.duration_ms} ms</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </Card>
            <Card title="قابلیت‌های مالی قانون‌مند (قفل)">
              <div className="v-banner v-banner--warning" style={{ marginBlockEnd: 12 }}>
                این قابلیت‌ها تا عبور از دروازه‌های تأیید حقوقی، امنیتی و تأمین‌کننده خاموش و قفل‌اند و از این پنل فعال نمی‌شوند.
              </div>
              <table className="v-table">
                <thead><tr><th>قابلیت</th><th>دروازه</th><th>وضعیت</th></tr></thead>
                <tbody>
                  {[...data.regulated_capabilities].sort((a, b) => a.key.localeCompare(b.key)).map((c) => (
                    <tr key={c.key}>
                      <td>
                        <div>{gateLabel[c.key] ?? c.key}</div>
                        <div style={{ font: "var(--type-caption)", color: "var(--color-on-surface-variant)" }}><span className="ltr">{c.key}</span></div>
                      </td>
                      <td><Badge tone="brand">{c.gate}</Badge></td>
                      <td><Badge tone="danger" icon="lock">قفل</Badge></td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </Card>
          </div>
        </>
      )}
    </>
  );
}
