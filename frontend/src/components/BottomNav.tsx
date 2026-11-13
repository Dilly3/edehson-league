import type { MouseEvent, ReactNode } from "react";
import { createPortal } from "react-dom";

export type BottomNavItem = { key: string; label: string; icon: ReactNode; active: boolean; href?: string; onSelect: () => void };

/** Sticky icon bar fixed to the bottom of the viewport.
 * Rendered into <body> so animated or transformed ancestors (e.g. `.page > *`)
 * cannot become its containing block and break `position: fixed`. */
export default function BottomNav({ items, label }: { items: BottomNavItem[]; label: string }) {
  return createPortal(
    <nav className="bottom-nav" aria-label={label}>
      {items.map(({ key, label, icon, active, href, onSelect }) => {
        const content = <>
          <span className="bottom-nav-icon" aria-hidden="true">{icon}</span>
          <span className="bottom-nav-label">{label}</span>
        </>;
        const className = active ? "bottom-nav-item active" : "bottom-nav-item";
        return href
          ? <a key={key} href={href} className={className} aria-current={active ? "page" : undefined}
              onClick={(event: MouseEvent) => { event.preventDefault(); onSelect(); }}>{content}</a>
          : <button key={key} type="button" className={className} aria-current={active ? "true" : undefined}
              onClick={onSelect}>{content}</button>;
      })}
    </nav>,
    document.body,
  );
}
