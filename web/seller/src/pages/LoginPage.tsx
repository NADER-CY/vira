import { Badge, Card, ErrorState, FeatureDisabledState, LoadingState, ThemeToggle, useAsync, type ClientConfig } from "@vira/shared";
import { publicApi } from "../api";

/** Seller sign-in. The web portal is controlled by the non-regulated flag seller.web_portal; the OTP sign-in
 * flow with the shared VIRA identity and store selection is built in Phase 4 (doc 04 §10). */
export function LoginPage() {
  const { data, error, loading, reload } = useAsync((signal) => publicApi.get<ClientConfig>("/v1/config", { signal }).then((r) => r.data), []);
  const enabled = data?.flags?.["seller.web_portal"] === true;
  return (
    <div className="v-auth">
      <div style={{ position: "fixed", insetBlockStart: 16, insetInlineEnd: 16 }}><ThemeToggle /></div>
      <div className="v-auth__card">
        <div style={{ display: "flex", alignItems: "center", gap: 12 }}>
          <span className="v-brand__mark" aria-hidden>و</span>
          <div>
            <h1 className="v-auth__title">پنل فروشندگان ویرا</h1>
            <p className="v-auth__sub">مدیریت غرفه از وب، با همان حساب کاربری اپلیکیشن ویرا</p>
          </div>
        </div>
        <Card>
          {loading && !data ? <LoadingState rows={3} /> : error ? <ErrorState error={error} onRetry={reload} /> : enabled ? (
            <FeatureDisabledState title="ورود با کد یکبارمصرف" phase="فاز ۴" />
          ) : (
            <>
              <FeatureDisabledState title="ورود فروشندگان از وب هنوز فعال نشده است" phase="فاز ۴" />
              <div style={{ display: "flex", justifyContent: "center" }}><Badge tone="neutral">seller.web_portal: خاموش</Badge></div>
            </>
          )}
        </Card>
        <p className="v-auth__sub" style={{ font: "var(--type-caption)" }}>
          تا آن زمان، فروشندگان غرفهٔ خود را از بخش «مدیریت غرفهٔ من» در اپلیکیشن ویرا مدیریت می‌کنند.
        </p>
      </div>
    </div>
  );
}
