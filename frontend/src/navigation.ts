export type Page = "overview" | "standings" | "fixtures" | "players" | "teams" | "membership" | "admin";
const publicPages = ["overview", "standings", "fixtures", "players", "teams", "membership"];
export function isAdminPath(path: string) { return path === "/admin" || path === "/admin/"; }
export function resolvePage(path: string, hash: string): Page {
  const raw = hash.replace(/^#/, "");
  const page = raw === "league" ? "overview" : raw;
  if (isAdminPath(path)) return publicPages.includes(page) ? page as Page : "admin";
  return publicPages.includes(page) ? page as Page : "membership";
}
export function pageURL(page: Page, adminArea: boolean) {
  if (page === "admin") return "/admin";
  return `${adminArea ? "/admin" : "/"}#${page}`;
}
