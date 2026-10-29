import { Navigate, Route, Routes } from "react-router";
import { AppShell, Button, EmptyState, ProtectedRoute, type NavGroup } from "@vira/shared";
import { useSellerAuth } from "./auth";
import { LoginPage } from "./pages/LoginPage";
import { OverviewPage } from "./pages/OverviewPage";

// Seller Admin information architecture (doc 09 §4.3). Same backend and permissions as the app's
// “Manage My Stall”; sections become active in the phases that build them.
export const sellerNav: NavGroup[] = [
  { items: [{ label: "نمای کلی فروشگاه", icon: "dashboard", to: "/" }] },
  { title: "فروش", items: [
    { label: "محصولات", icon: "inventory_2", to: "/products", phase: "فاز ۴" },
    { label: "موجودی انبار", icon: "sell", to: "/inventory", phase: "فاز ۴" },
    { label: "سفارش‌ها و ارسال", icon: "local_shipping", to: "/orders", phase: "فاز ۷" },
    { label: "مرجوعی‌ها", icon: "assignment_return", to: "/returns", phase: "فاز ۷" },
    { label: "پیام‌های مشتریان", icon: "chat", to: "/inbox", phase: "فاز ۵" },
  ] },
  { title: "مالی", items: [
    { label: "صورت‌حساب و فاکتورها", icon: "receipt_long", to: "/statements", phase: "فاز ۷" },
    { label: "تسویه و برداشت", icon: "payments", to: "/payouts", phase: "FG-1b", locked: true },
    { label: "برداشت USDT", icon: "account_balance_wallet", to: "/usdt", phase: "G2", locked: true },
  ] },
  { title: "فروشگاه", items: [
    { label: "تخفیف‌ها و کمپین‌ها", icon: "campaign", to: "/promotions", phase: "فاز ۹" },
    { label: "کارکنان و دسترسی‌ها", icon: "group", to: "/staff", phase: "فاز ۴" },
    { label: "تنظیمات فروشگاه", icon: "settings", to: "/settings", phase: "فاز ۴" },
  ] },
];

function Shell() {
  const { principal, signOut } = useSellerAuth();
  return (
    <AppShell brand="ویرا" subtitle="پنل فروشندگان" nav={sellerNav}
      topbar={<span className="v-topbar__spacer" />}
      user={
        <div className="v-user">
          <span className="v-avatar" aria-hidden>ف</span>
          <span className="ltr" style={{ font: "var(--type-caption)" }}>{principal?.subject}</span>
          <Button variant="ghost" icon="logout" aria-label="خروج" title="خروج" onClick={signOut} />
        </div>
      }>
      <Routes>
        <Route index element={<OverviewPage />} />
        <Route path="*" element={<EmptyState icon="search" title="صفحه پیدا نشد" />} />
      </Routes>
    </AppShell>
  );
}

export function App() {
  const { status } = useSellerAuth();
  return (
    <Routes>
      <Route path="/login" element={status === "authenticated" ? <Navigate to="/" replace /> : <LoginPage />} />
      <Route path="/*" element={<ProtectedRoute status={status}><Shell /></ProtectedRoute>} />
    </Routes>
  );
}
