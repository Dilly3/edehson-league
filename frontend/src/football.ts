import type { Fixture, MatchEvent, Player, PlayerStats } from "./types";
export const eventLabels = {
  goal: "Goal",
  own_goal: "Own goal",
  yellow_card: "Yellow card",
  red_card: "Red card",
};
export function scoreEvents(
  fixture: Fixture,
  events: MatchEvent[],
  rosters: Record<string, Player[]>,
) {
  let home = 0,
    away = 0;
  for (const event of events) {
    if (event.type !== "goal" && event.type !== "own_goal") continue;
    const homePlayer = (rosters[fixture.home_team_tag] ?? []).some(
      (p) => p.id === event.player_id,
    );
    const awayPlayer = (rosters[fixture.away_team_tag] ?? []).some(
      (p) => p.id === event.player_id,
    );
    const team = event.player_id ? (homePlayer?fixture.home_team_tag:awayPlayer?fixture.away_team_tag:undefined) : event.team_tag;
    if (team!==fixture.home_team_tag&&team!==fixture.away_team_tag) continue;
    const isHome=team===fixture.home_team_tag;
    const creditHome = event.type === "own_goal" ? !isHome : isHome;
    if (creditHome) home++;
    else away++;
  }
  return { home, away };
}
export function validateEvents(events: MatchEvent[]): string | null {
  let lastMinute = -1;
  const cards = new Map<string, { yellow: number; red: number }>();
  for (let i = 0; i < events.length; i++) {
    const e = events[i];
    const prefix = `Event ${i + 1}: `;
    if (!e.player_id) {
      if(e.type!=="goal"&&e.type!=="own_goal")return prefix+"select a player.";
      if(!e.team_tag)return prefix+"select a team for the unnamed goal.";
    }
    if (e.minute !== null) {
      if (!Number.isInteger(e.minute) || e.minute < 0)
        return prefix + "enter a nonnegative whole minute.";
      if (e.minute < lastMinute)
        return prefix + "events must be in match order.";
      lastMinute = e.minute;
    }
    if (!Object.hasOwn(eventLabels, e.type))
      return prefix + "unsupported event type.";
    const count = cards.get(e.player_id) ?? { yellow: 0, red: 0 };
    if (e.type === "yellow_card" && ++count.yellow > 2)
      return (
        prefix + "a player cannot receive more than two yellow cards per match."
      );
    if (e.type === "red_card" && ++count.red > 1)
      return (
        prefix + "a player cannot receive more than one red card per match."
      );
    cards.set(e.player_id, count);
  }
  return null;
}
export function formatKickoff(value: string | null) {
  return value
    ? new Intl.DateTimeFormat("en-NG", {
        timeZone: "Africa/Lagos",
        day: "numeric",
        month: "short",
        hour: "2-digit",
        minute: "2-digit",
        hour12: true,
      }).format(new Date(value))
    : "Kickoff to be confirmed";
}
export function kickoffParts(value: string | null) {
  if (!value) return null;
  const when = new Date(value);
  const zone = { timeZone: "Africa/Lagos" } as const;
  return {
    date: new Intl.DateTimeFormat("en-NG", {
      ...zone,
      weekday: "short",
      day: "numeric",
      month: "short",
    }).format(when),
    time: new Intl.DateTimeFormat("en-GB", {
      ...zone,
      hour: "2-digit",
      minute: "2-digit",
      hour12: false,
    }).format(when),
  };
}
export function kickoffToInput(value: string | null) {
  if (!value) return "";
  const d = new Date(new Date(value).getTime() + 3600000);
  return d.toISOString().slice(0, 16);
}
export function kickoffFromInput(value: string) {
  return value ? new Date(`${value}:00+01:00`).toISOString() : null;
}

export function venueFromInput(value: string): string | null {
  return value.trim() || null;
}

// The app has no live-match feed: elapsed scheduled fixtures await a result.
export function awaitingResult(fixture: Fixture, now = Date.now()) {
  return fixture.status === "scheduled" && fixture.kickoff_at !== null &&
    Number.isFinite(Date.parse(fixture.kickoff_at)) && Date.parse(fixture.kickoff_at) <= now;
}

export function teamInitials(name: string) {
  const words = name.trim().split(/[\s\-–—_]+/).filter(Boolean);
  if (words.length > 1) return (words[0][0] + words[words.length - 1][0]).toUpperCase();
  return Array.from(words[0] ?? "?").slice(0, 2).join("").toUpperCase();
}

export function scheduleSummary(fixtures: Fixture[], teamCount: number) {
  const expected = teamCount * Math.max(0, teamCount - 1);
  const gamesPerDay = Math.floor(teamCount / 2);
  const days = new Map<number, number>();
  for (const f of fixtures) days.set(f.matchday, (days.get(f.matchday) ?? 0) + 1);
  const expectedDays = teamCount < 2 ? 0 : (teamCount % 2 ? teamCount : teamCount - 1) * 2;
  const incomplete = Array.from({ length: expectedDays }, (_, i) => i + 1)
    .filter(day => (days.get(day) ?? 0) !== gamesPerDay);
  const unexpectedDays = [...days.keys()].filter(day => day < 1 || day > expectedDays).sort((a,b) => a-b);
  return { expected, expectedDays, incomplete, unexpectedDays };
}

export function matchStatusLabel(status: string) {
  return ({scheduled: "Scheduled", live: "Live", half_time: "Half-time", finalized: "Full-time", postponed: "Postponed", abandoned: "Abandoned"} as Record<string, string>)[status] ?? status.replaceAll("_", " ");
}
export function hasPublishedScore(fixture: Fixture) {
  return ["live", "half_time", "finalized"].includes(fixture.status) && fixture.home_score !== null && fixture.away_score !== null;
}

export function teamSeasonSummary(teamTag: string, stats: PlayerStats[]) {
  const players = stats.filter(player => player.team_tag === teamTag);
  const goals = players.reduce((max, player) => Math.max(max, player.goals), 0);
  return {
    goals,
    scorers: goals > 0 ? players.filter(player => player.goals === goals).sort((a,b) => a.name.localeCompare(b.name)) : [],
    yellowCards: players.reduce((total, player) => total + player.yellow_cards, 0),
    redCards: players.reduce((total, player) => total + player.red_cards, 0),
  };
}

export function defenceSummary(played: number, conceded: number) {
  if (played === 0) return "No matches played yet, so there is no defensive record to assess.";
  return `They have conceded ${conceded} ${conceded === 1 ? "goal" : "goals"} in ${played} ${played === 1 ? "match" : "matches"} played — ${(conceded / played).toFixed(1)} per match.`;
}

// Use the admin's short display name when present; otherwise preserve the full name.
export function eventTeamLabel(name: string, displayName?: string | null) {
  return displayName?.trim() || name;
}
