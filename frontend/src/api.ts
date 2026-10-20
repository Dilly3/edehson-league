import type { Player, PlayerStats } from "./types";
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
  const timeout = setTimeout(() => controller.abort(), 20000);
  try {
    const headers = new Headers(options.headers);
    headers.set("Accept", "application/json");
    if (options.body) headers.set("Content-Type", "application/json");
    if (token) headers.set("Authorization", `Bearer ${token}`);
    const response = await fetch(`/api${path}`, {
      ...options,
      headers,
      signal: controller.signal,
      cache: "no-store",
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
