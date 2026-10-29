import { Card, FeatureDisabledState, PageHeader } from "@vira/shared";

export function OverviewPage() {
  return (
    <>
      <PageHeader title="نمای کلی فروشگاه" subtitle="فروش، سفارش‌ها و وضعیت غرفه — از فاز ۴" />
      <div className="v-grid">
        <Card title="سفارش‌های امروز"><FeatureDisabledState title="سفارش‌ها" phase="فاز ۷" /></Card>
        <Card title="محصولات فعال"><FeatureDisabledState title="کاتالوگ" phase="فاز ۴" /></Card>
        <Card title="تسویه"><FeatureDisabledState title="تسویهٔ زنده" phase="FG-1b" locked /></Card>
      </div>
    </>
  );
}
