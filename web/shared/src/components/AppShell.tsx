import type { ReactNode } from "react";
import { NavLink } from "react-router";
import { Icon, type IconName } from "./Icon";
import { Badge, ThemeToggle } from "./ui";

export interface NavItem { label: string; icon: IconName; to: string; phase?: string; locked?: boolean }
export interface NavGroup { title?: string; items: NavItem[] }

/** Admin-style shell: sidebar on the start (right) side, sticky top bar (layout of the archive admin consoles). */
export function AppShell({ brand, subtitle, nav, user, topbar, children }: {
  brand: string; subtitle: string; nav: NavGroup[]; user?: ReactNode; topbar?: ReactNode; children: ReactNode;
}) {
  return (
    <div className="v-shell">
      <aside className="v-sidebar" aria-label="ناوبری اصلی">
        <div className="v-brand">
          <span className="v-brand__mark" aria-hidden>و</span>
          <div><div className="v-brand__name">{brand}</div><div className="v-brand__sub">{subtitle}</div></div>
        </div>
        <nav style={{ display: "grid", gap: 12 }}>
          {nav.map((g, gi) => (
            <div className="v-nav-group" key={gi}>
              {g.title && <div className="v-nav-group__title">{g.title}</div>}
              {g.items.map((it) => it.phase ? (
                <span key={it.to} className="v-nav-item" aria-disabled="true" title={it.locked ? `قفل تا ${it.phase}` : `در ${it.phase}`}>
                  <Icon name={it.icon} />{it.label}
                  <span className="v-nav-item__phase"><Badge tone={it.locked ? "danger" : "neutral"} icon={it.locked ? "lock" : undefined}>{it.phase}</Badge></span>
                </span>
              ) : (
                <NavLink key={it.to} to={it.to} end={it.to === "/"} className="v-nav-item"><Icon name={it.icon} />{it.label}</NavLink>
              ))}
            </div>
          ))}
        </nav>
      </aside>
      <div className="v-main">
        <header className="v-topbar">
          {topbar ?? <span className="v-topbar__spacer" />}
          <ThemeToggle />
          {user}
        </header>
        <main className="v-content" id="main">{children}</main>
      </div>
    </div>
  );
}
