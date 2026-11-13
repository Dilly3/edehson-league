import type { ReactNode } from "react";

export type HeaderNavItem = { key: string; label: string; icon: ReactNode; href: string; active: boolean; onSelect: () => void };

/** Compact icon navigation inside the sticky site header. Icon with label beside it on wide screens, stacked below it on phones. */
export default function HeaderNav({ items }: { items: HeaderNavItem[] }) {
  return (
    <nav className="header-nav" aria-label="Main navigation">
      {items.map(({ key, label, icon, href, active, onSelect }) => (
        <a key={key} href={href} className={active ? "header-nav-item active" : "header-nav-item"}
            aria-current={active ? "page" : undefined}
            onClick={event => { event.preventDefault(); onSelect(); }}>
            <span className="header-nav-icon" aria-hidden="true">{icon}</span>
            <span className="header-nav-label">{label}</span>
        </a>
      ))}
    </nav>
  );
}
