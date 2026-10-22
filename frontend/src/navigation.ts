export type Page = "overview" | "standings" | "fixtures" | "players" | "teams" | "admin";
const publicPages = ["overview", "standings", "fixtures", "players", "teams"];
export function isAdminPath(path: string) { return path === "/admin" || path === "/admin/"; }
export function resolvePage(path: string, hash: string): Page {
  const page = hash.replace(/^#/, "");
  if (isAdminPath(path)) return publicPages.includes(page) ? page as Page : "admin";
  return publicPages.includes(page) ? page as Page : "overview";
}
export function pageURL(page: Page, adminArea: boolean) {
  if (page === "admin") return "/admin";
  return `${adminArea ? "/admin" : "/"}#${page}`;
}
