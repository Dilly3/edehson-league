import type { Player, PlayerStats } from "./types";
let adminCredential="";
export function setAdminCredential(value:string){adminCredential=value;}
/** Pass as the token to send a request without any admin credential (e.g. the login call itself). */
const NO_CREDENTIAL=" ";
export class ApiError extends Error {
  status: number;
  constructor(message: string, status: number) {
    super(message);
    this.status = status;
  }
}
export async function api<T>(
  path: string,
  options: RequestInit = {},
  token?: string,
): Promise<T> {
  const controller = new AbortController();
  const upload = typeof FormData !== "undefined" && options.body instanceof FormData;
  // Receipt uploads can be several MB on a phone connection; give them longer.
  const timeout = setTimeout(() => controller.abort(), upload ? 90000 : 20000);
  try {
    const headers = new Headers(options.headers);
    headers.set("Accept", "application/json");
    if (options.body && !upload) headers.set("Content-Type", "application/json");
    const credential=token===NO_CREDENTIAL?"":(token||adminCredential);
    if (credential) headers.set("Authorization", `Bearer ${credential}`);
    const response = await fetch(`/api${path}`, {
      ...options,
      headers,
      signal: controller.signal,
      cache: "no-store",
      credentials: "same-origin",
    });
    const text = await response.text();
    let body: Record<string, unknown> = {};
    try {
      body = text ? JSON.parse(text) : {};
    } catch {
      throw new ApiError(
        "The API returned an unexpected response. Check that the Go backend is running.",
        response.status,
      );
    }
    if (!response.ok)
      throw new ApiError(
        typeof body.error === "string"
          ? body.error
          : response.status >= 500
            ? "The league API is unavailable. Check that the Go backend is running on port 8011."
            : "The request could not be completed.",
        response.status,
      );
    return (Object.hasOwn(body, "data") ? body.data : body) as T;
  } catch (error) {
    if (error instanceof ApiError) throw error;
    if (error instanceof Error && error.name === "AbortError")
      throw new Error("The API took too long to respond. Please try again.");
    throw new Error(
      "Cannot reach the league API. Check that your backend is running on port 8011.",
    );
  } finally {
    clearTimeout(timeout);
  }
}
// Support current snake_case roster DTOs and older capitalized responses.
export async function getPlayers(tag: string): Promise<Player[]> {
  const rows = await api<Record<string, unknown>[]>(
    `/teams/${encodeURIComponent(tag)}/players`,
  );
  return rows.map((p) => {
    const jersey = String(p.jersey_number ?? p.JerseyNumber ?? "");
    const teamTag = String(p.team_tag ?? p.TeamTag ?? tag);
    return {
      id: String(p.id ?? p.ID ?? ""),
      name: String(p.name ?? p.Name ?? ""),
      jersey_number: jersey,
      team: String(p.team ?? p.Team ?? ""),
      team_tag: teamTag,
      player_tag: String(p.player_tag ?? p.PlayerTag ?? `${teamTag}-${jersey}`),
      is_captain: (p.is_captain ?? p.IsCaptain) === true,
    };
  }).sort((a, b) => {
    const number = (jersey: string) => /^\d+$/.test(jersey.trim()) ? Number(jersey) : Infinity;
    return number(a.jersey_number) - number(b.jersey_number) || a.name.localeCompare(b.name);
  });
}
export const message = (error: unknown) =>
  error instanceof Error
    ? error.message
    : "Something went wrong. Please try again.";

function rosterPlayerPath(teamTag: string, playerID: string) {
  return `/teams/${encodeURIComponent(teamTag)}/players/${encodeURIComponent(playerID)}`;
}
export function updatePlayer(
  teamTag: string,
  playerID: string,
  input: { name: string; jersey_number: number },
  token: string,
) {
  return api<void>(
    rosterPlayerPath(teamTag, playerID),
    { method: "PUT", body: JSON.stringify(input) },
    token,
  );
}
export function assignCaptain(
  teamTag: string,
  playerID: string,
  token: string,
) {
  return api<void>(
    `${rosterPlayerPath(teamTag, playerID)}/captain`,
    { method: "PUT" },
    token,
  );
}
export function removePlayer(teamTag: string, playerID: string, token: string) {
  return api<void>(
    rosterPlayerPath(teamTag, playerID),
    { method: "DELETE" },
    token,
  );
}

// Support the current API and the simplified red_cards field without showing old disciplinary fields.
export async function getPlayerStats(): Promise<PlayerStats[]> {
  const rows = await api<Record<string, unknown>[]>("/player-stats");
  return rows.map((p) => ({
    player_id: String(p.player_id ?? ""),
    player_tag: String(p.player_tag ?? ""),
    name: String(p.name ?? ""),
    team_tag: String(p.team_tag ?? ""),
    team_name: String(p.team_name ?? ""),
    jersey_number: String(p.jersey_number ?? ""),
    goals: Number(p.goals ?? 0),
    own_goals: p.own_goals == null ? undefined : Number(p.own_goals),
    yellow_cards: Number(p.yellow_cards ?? 0),
    red_cards: Number(p.red_cards ?? p.straight_red_cards ?? 0),
  }));
}
export function updateFixtureVenue(
  fixtureID: string,
  venue: string | null,
  token: string,
) {
  return api<void>(
    `/fixtures/${encodeURIComponent(fixtureID)}/venue`,
    {
      method: "PATCH",
      body: JSON.stringify({ venue }),
    },
    token,
  );
}

export type AdminRole = "super_admin" | "admin";
export interface AdminPermissions { league: boolean; roaster: boolean }
export interface AdminSession { token: string; role: AdminRole; username: string; permissions: AdminPermissions }
export interface AdminUser { id: string; username: string; active: boolean; can_league: boolean; can_roaster: boolean; created_at: string; last_login_at: string | null }

/** Super admin: verifies the ADMIN_TOKEN key. */
export async function verifyAdminKey(key: string): Promise<AdminSession> {
  const result = await api<{ authenticated: boolean; role: AdminRole; username: string; permissions: AdminPermissions }>("/admin/session", {method:"GET"}, key);
  if (result.authenticated !== true) throw new Error("Admin authentication could not be verified.");
  return { token: key, role: result.role, username: result.username, permissions: result.permissions };
}

/** Admin account: exchanges username and password for a session token. */
export function loginAdmin(username: string, password: string): Promise<AdminSession> {
  return api<AdminSession>("/admin/login", { method: "POST", body: JSON.stringify({ username, password }) }, NO_CREDENTIAL);
}

export async function logoutAdmin(token: string): Promise<void> {
  await api("/admin/logout", { method: "POST", body: "{}" }, token);
}

export const listAdmins = (token: string) => api<AdminUser[]>("/admin/users", {}, token);
export const createAdmin = (token: string, username: string, password: string, perms: AdminPermissions) =>
  api<AdminUser>("/admin/users", { method: "POST", body: JSON.stringify({ username, password, can_league: perms.league, can_roaster: perms.roaster }) }, token);
export const updateAdmin = (token: string, id: string, change: { active?: boolean; password?: string; can_league?: boolean; can_roaster?: boolean }) =>
  api<AdminUser>(`/admin/users/${encodeURIComponent(id)}`, { method: "PATCH", body: JSON.stringify(change) }, token);
export const deleteAdmin = (token: string, id: string) =>
  api<void>(`/admin/users/${encodeURIComponent(id)}`, { method: "DELETE" }, token);
