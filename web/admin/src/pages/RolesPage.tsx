import { Badge, Card, ErrorState, LoadingState, PageHeader, useAsync, type Role } from "@vira/shared";
import { adminApi } from "../api";

/** Admin roles and permissions (RBAC, ADR-017). Read-only in Phase 1; role assignment UI comes with staff management. */
export function RolesPage() {
  const { data, error, loading, reload } = useAsync((signal) => adminApi.get<{ data: Role[] }>("/v1/admin/roles", { signal }).then((r) => r.data.data), []);
  return (
    <>
      <PageHeader title="نقش‌ها و دسترسی‌ها" subtitle="دسترسی‌ها به نقش داده می‌شوند و هر درخواست سمت سرور بررسی می‌شود." />
      {loading && !data ? <Card><LoadingState /></Card> : error ? <Card><ErrorState error={error} onRetry={reload} /></Card> : (
        <div className="v-grid" style={{ gridTemplateColumns: "repeat(auto-fit, minmax(300px, 1fr))" }}>
          {data?.map((r) => (
            <Card key={r.key} title={<>{r.name_fa} <span className="ltr" style={{ font: "var(--type-caption)", color: "var(--color-on-surface-variant)" }}>{r.key}</span></>}>
              {r.description && <p style={{ color: "var(--color-on-surface-variant)", marginBlockEnd: 12 }}>{r.description}</p>}
              <div style={{ display: "flex", flexWrap: "wrap", gap: 6 }}>
                {r.permissions.map((p) => <Badge key={p} tone="info"><span className="ltr">{p}</span></Badge>)}
              </div>
            </Card>
          ))}
        </div>
      )}
    </>
  );
}
