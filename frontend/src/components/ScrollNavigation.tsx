import { useEffect, useRef, useState, type ReactNode } from "react";
import ChevronLeftRounded from "@mui/icons-material/ChevronLeftRounded";
import ChevronRightRounded from "@mui/icons-material/ChevronRightRounded";

export default function ScrollNavigation({ children }: { children: ReactNode }) {
  const ref = useRef<HTMLElement>(null);
  const [edges, setEdges] = useState({ left: false, right: false });

  useEffect(() => {
    const nav = ref.current;
    if (!nav) return;
    const update = () => setEdges({
      left: nav.scrollLeft > 2,
      right: nav.scrollLeft + nav.clientWidth < nav.scrollWidth - 2,
    });
    const observer = new ResizeObserver(update);
    observer.observe(nav);
    Array.from(nav.children).forEach(child => observer.observe(child));
    const revealActive = () => {
      const active = nav.querySelector<HTMLElement>('[aria-current="page"]');
      if (!active) return;
      const bounds = nav.getBoundingClientRect();
      const item = active.getBoundingClientRect();
      if (item.left < bounds.left) nav.scrollLeft += item.left - bounds.left;
      else if (item.right > bounds.right) nav.scrollLeft += item.right - bounds.right;
      update();
    };
    const selectionObserver = new MutationObserver(revealActive);
    selectionObserver.observe(nav, { subtree: true, attributes: true, attributeFilter: ["aria-current"] });
    window.addEventListener("resize", revealActive);
    nav.addEventListener("scroll", update, { passive: true });
    revealActive();
    return () => {
      observer.disconnect();
      selectionObserver.disconnect();
      window.removeEventListener("resize", revealActive);
      nav.removeEventListener("scroll", update);
    };
  }, []);

  const scroll = (direction: number) => {
    const nav = ref.current;
    if (!nav) return;
    nav.scrollBy({ left: direction * nav.clientWidth * 0.7,
      behavior: window.matchMedia("(prefers-reduced-motion: reduce)").matches ? "instant" : "smooth" });
  };

  return (
    <div className="navigation-shell">
      <button type="button" className="navigation-arrow" disabled={!edges.left}
        aria-label="Scroll menu left" aria-controls="main-navigation" onClick={() => scroll(-1)}>
        <ChevronLeftRounded />
      </button>
      <nav ref={ref} id="main-navigation" aria-label="Main navigation">{children}</nav>
      <button type="button" className="navigation-arrow" disabled={!edges.right}
        aria-label="Scroll menu right" aria-controls="main-navigation" onClick={() => scroll(1)}>
        <ChevronRightRounded />
      </button>
    </div>
  );
}
