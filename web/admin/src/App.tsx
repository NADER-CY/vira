import { Navigate, Route, Routes, useLocation } from "react-router";
import { AppShell, Button, EmptyState, ProtectedRoute, type NavGroup } from "@vira/shared";
import { useAuth } from "./auth";
import { LoginPage } from "./pages/LoginPage";
import { DashboardPage } from "./pages/DashboardPage";
import { FlagsPage } from "./pages/FlagsPage";
import { AuditPage } from "./pages/AuditPage";
import { RolesPage } from "./pages/RolesPage";
import { UsersPage } from "./pages/UsersPage";
import { ModerationPage } from "./pages/ModerationPage";

// Single sidebar replacing the archive's 46 variants (doc 09 §4.2). Items of later phases are visible but
// inactive and labelled with their phase or gate — nothing on the roadmap is silently removed.
export const adminNav: NavGroup[] = [
  { items: [{ label: "داشبورد", icon: "dashboard", to: "/" }] },
  { title: "کاربران و اعتماد", items: [
    { label: "کاربران", icon: "group", to: "/users" },
    { label: "احراز هویت و KYC", icon: "verified_user", to: "/kyc", phase: "فاز ۴" },
    { label: "فروشندگان", icon: "storefront", to: "/sellers", phase: "فاز ۴" },
  ] },
  { title: "محتوا", items: [
    { label: "پست‌ها و ویدیوها", icon: "video_library", to: "/content", phase: "فاز ۳" },
    { label: "گزارش‌ها و نظرات", icon: "block", to: "/moderation" },
  ] },
  { title: "تجارت", items: [
    { label: "محصولات و دسته‌ها", icon: "inventory_2", to: "/catalog", phase: "فاز ۴" },
    { label: "سفارش‌ها", icon: "shopping_bag", to: "/orders", phase: "فاز ۷" },
    { label: "ارسال", icon: "local_shipping", to: "/shipping", phase: "فاز ۷" },
    { label: "مرجوعی و اختلاف", icon: "assignment_return", to: "/returns", phase: "فاز ۷" },
  ] },
  { title: "مالی", items: [
    { label: "پرداخت‌ها و تطبیق", icon: "payments", to: "/payments", phase: "FG-1b", locked: true },
    { label: "تسویه و برداشت", icon: "receipt_long", to: "/settlements", phase: "FG-1b", locked: true },
    { label: "کیف پول", icon: "account_balance_wallet", to: "/wallet", phase: "G1", locked: true },
    { label: "USDT و تبدیل", icon: "account_balance_wallet", to: "/usdt", phase: "G2", locked: true },
  ] },
  { title: "رشد", items: [
    { label: "تبلیغات و کمپین‌ها", icon: "campaign", to: "/growth", phase: "فاز ۹" },
    { label: "هوش مصنوعی", icon: "smart_toy", to: "/ai", phase: "فاز ۹" },
  ] },
  { title: "پلتفرم", items: [
    { label: "فیچر فلگ‌ها", icon: "flag", to: "/flags" },
    { label: "گزارش ممیزی", icon: "history", to: "/audit" },
    { label: "نقش‌ها و دسترسی‌ها", icon: "admin_panel_settings", to: "/roles" },
  ] },
];

function Shell() {
  const { admin, logout } = useAuth();
  const initial = admin?.display_name?.trim()?.[0] ?? "؟";
  return (
    <AppShell
      brand="ویرا"
      subtitle="مدیریت ارشد پلتفرم"
      nav={adminNav}
      topbar={<span className="v-topbar__spacer" />}
      user={
        <div className="v-user">
          <span className="v-avatar" aria-hidden>{initial}</span>
          <div style={{ display: "grid" }}>
            <strong>{admin?.display_name}</strong>
            <span className="ltr" style={{ font: "var(--type-caption)", color: "var(--color-on-surface-variant)" }}>{admin?.email}</span>
          </div>
          <Button variant="ghost" icon="logout" aria-label="خروج" title="خروج" onClick={() => void logout()} />
        </div>
      }
    >
      <Routes>
        <Route index element={<DashboardPage />} />
        <Route path="users" element={<UsersPage />} />
        <Route path="moderation" element={<ModerationPage />} />
        <Route path="flags" element={<FlagsPage />} />
        <Route path="audit" element={<AuditPage />} />
        <Route path="roles" element={<RolesPage />} />
        <Route path="*" element={<EmptyState icon="search" title="صفحه پیدا نشد" description="این بخش وجود ندارد یا هنوز فعال نشده است." />} />
      </Routes>
    </AppShell>
  );
}

/** After sign-in, return to the page that required it (state set by ProtectedRoute). */
function AfterLogin() {
  const from = (useLocation().state as { from?: string } | null)?.from;
  return <Navigate to={from && from !== "/login" ? from : "/"} replace />;
}

export function App() {
  const { status } = useAuth();
  return (
    <Routes>
      <Route path="/login" element={status === "authenticated" ? <AfterLogin /> : <LoginPage />} />
      <Route path="/*" element={<ProtectedRoute status={status}><Shell /></ProtectedRoute>} />
    </Routes>
  );
}
