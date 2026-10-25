import { describe, it, expect } from "vitest";
import {
  awaitingResult,
  defenceSummary,
  teamSeasonSummary,
  matchStatusLabel,
  hasPublishedScore,
  teamInitials,
  eventTeamLabel,
  scheduleSummary,
  kickoffFromInput,
  kickoffToInput,
  scoreEvents,
  validateEvents,
  venueFromInput,
} from "./football";
import type { Fixture, MatchEvent, Player, PlayerStats } from "./types";
const fixture = { home_team_tag: "EDL001", away_team_tag: "EDL002" } as Fixture;
const rosters = {
  EDL001: [{ id: "home" } as Player],
  EDL002: [{ id: "away" } as Player],
};
const e = (
  player_id: string,
  type: MatchEvent["type"],
  minute: number | null,
): MatchEvent => ({ player_id, type, minute });
describe("result editor", () => {
  it("credits own goals to the opponent and excludes cards from scores", () => {
    expect(
      scoreEvents(
        fixture,
        [
          e("home", "goal", 10),
          e("home", "own_goal", 20),
          e("away", "yellow_card", 30),
        ],
        rosters,
      ),
    ).toEqual({ home: 1, away: 1 });
  });
  it("requires players for cards and a team for unnamed goals", () => {
    expect(validateEvents([e("", "yellow_card", 10)])).toContain("select a player");
    expect(validateEvents([e("", "red_card", 10)])).toContain("select a player");
    expect(validateEvents([e("", "goal", 10)])).toContain("select a team");
    expect(validateEvents([{...e("", "goal", 10),team_tag:"EDL001"}])).toBeNull();
  });
  it("accepts two yellow-card events without inferring a dismissal", () => {
    expect(
      validateEvents([
        e("home", "yellow_card", 10),
        e("home", "yellow_card", 30),
        e("home", "goal", 40),
      ]),
    ).toBeNull();
  });
  it("rejects a third yellow for the same player", () => {
    expect(
      validateEvents([
        e("home", "yellow_card", 10),
        e("home", "yellow_card", 30),
        e("home", "yellow_card", 40),
      ]),
    ).toContain("more than two yellow");
  });
  it("counts cards separately for each player", () => {
    expect(
      validateEvents([
        e("home", "yellow_card", 10),
        e("home", "yellow_card", 20),
        e("away", "yellow_card", 30),
        e("away", "yellow_card", 40),
      ]),
    ).toBeNull();
  });
  it("allows goals after a red-card event", () => {
    expect(
      validateEvents([e("home", "red_card", 10), e("home", "goal", 20)]),
    ).toBeNull();
  });
  it("rejects more than one red card for the same player", () => {
    expect(
      validateEvents([e("home", "red_card", 10), e("home", "red_card", 20)]),
    ).toContain("more than one red");
  });
  it("does not create a red card when two yellow events are validated", () => {
    const events = [e("home", "yellow_card", 10), e("home", "yellow_card", 20)];
    expect(validateEvents(events)).toBeNull();
    expect(events).toHaveLength(2);
    expect(events.every((event) => event.type === "yellow_card")).toBe(true);
  });
  it("rejects decreasing known minutes", () => {
    expect(
      validateEvents([e("home", "goal", 30), e("away", "goal", 20)]),
    ).toContain("match order");
  });
  it("allows unknown minutes without converting them to zero", () => {
    expect(
      validateEvents([e("home", "goal", null), e("away", "goal", 25)]),
    ).toBeNull();
  });
  it("rejects negative or fractional minutes", () => {
    expect(validateEvents([e("home", "goal", -1)])).toContain("whole minute");
    expect(validateEvents([e("home", "goal", 1.5)])).toContain("whole minute");
  });
});
describe("Nigeria kickoff conversions", () => {
  it("shows UTC instants in WAT independent of browser timezone", () => {
    expect(kickoffToInput("2026-10-25T15:00:00Z")).toBe("2026-10-25T16:00");
  });
  it("submits WAT input as the correct UTC instant", () => {
    expect(kickoffFromInput("2026-10-25T16:00")).toBe(
      "2026-10-25T15:00:00.000Z",
    );
  });
  it("preserves unconfirmed kickoff times", () => {
    expect(kickoffToInput(null)).toBe("");
    expect(kickoffFromInput("")).toBeNull();
  });
});

describe("venue input", () => {
  it("trims a venue before saving", () => {
    expect(venueFromInput("  Community Ground  ")).toBe("Community Ground");
  });
  it("clears an unconfirmed venue with explicit null", () => {
    expect(venueFromInput("   ")).toBeNull();
  });
});


describe("match centre presentation", () => {
  const now = Date.parse("2026-10-18T12:00:00Z");
  it("flags elapsed scheduled matches without inventing live results", () => {
    expect(awaitingResult({status: "scheduled", kickoff_at: "2026-10-01T06:00:00Z"} as Fixture, now)).toBe(true);
    expect(awaitingResult({status: "scheduled", kickoff_at: "2026-10-19T06:00:00Z"} as Fixture, now)).toBe(false);
    expect(awaitingResult({status: "scheduled", kickoff_at: null} as Fixture, now)).toBe(false);
    expect(awaitingResult({status: "finalized", kickoff_at: "2026-10-01T06:00:00Z"} as Fixture, now)).toBe(false);
  });
  it("distinguishes hyphenated and spaced team names", () => {
    expect(teamInitials("Team-A")).toBe("TA");
    expect(teamInitials("Team B")).toBe("TB");
    expect(teamInitials("Real Edehsons")).toBe("RE");
    expect(teamInitials("Arsenal")).toBe("AR");
  });
  it("identifies missing matchday slots in the four-team league", () => {
    const summary = scheduleSummary([{matchday: 1}, {matchday: 1}, {matchday: 2}] as Fixture[], 4);
    expect(summary.expected).toBe(12);
    expect(summary.expectedDays).toBe(6);
    expect(summary.incomplete).toEqual([2,3,4,5,6]);
  });
});

it("flags matchdays outside the six-round league", () => {
  expect(scheduleSummary([{matchday: 8}] as Fixture[], 4).unexpectedDays).toEqual([8]);
});

it("shows in-progress scores with distinct status labels", () => {
  expect(matchStatusLabel("half_time")).toBe("Half-time");
  expect(matchStatusLabel("finalized")).toBe("Full-time");
  for (const status of ["live", "half_time", "finalized"]) {
    expect(hasPublishedScore({status, home_score: 0, away_score: 0} as Fixture)).toBe(true);
    expect(awaitingResult({status, kickoff_at: "2020-01-01T00:00:00Z"} as Fixture)).toBe(false);
  }
  expect(hasPublishedScore({status: "scheduled", home_score: null, away_score: null} as Fixture)).toBe(false);
});

it("finds joint team leaders and totals cards without including another team", () => {
  const stats = [
    {team_tag: "A", name: "Zed", goals: 3, yellow_cards: 2, red_cards: 1},
    {team_tag: "A", name: "Alex", goals: 3, yellow_cards: 1, red_cards: 0},
    {team_tag: "B", name: "Other", goals: 9, yellow_cards: 8, red_cards: 4},
  ] as PlayerStats[];
  const result = teamSeasonSummary("A", stats);
  expect(result.scorers.map(p => p.name)).toEqual(["Alex", "Zed"]);
  expect(result.goals).toBe(3);
  expect(result.yellowCards).toBe(3);
  expect(result.redCards).toBe(1);
  expect(teamSeasonSummary("C", stats).scorers).toEqual([]);
  expect(teamSeasonSummary("A", [{team_tag: "A", goals: 0, own_goals: 2, yellow_cards: 0, red_cards: 0} as PlayerStats]).scorers).toEqual([]);
});

it("describes defence from completed results without inventing form for unplayed teams", () => {
  expect(defenceSummary(0, 0)).toContain("no defensive record");
  expect(defenceSummary(2, 3)).toContain("3 goals in 2 matches played — 1.5 per match");
  expect(defenceSummary(1, 0)).toContain("0 goals in 1 match played");
});

describe("match event team labels", () => {
  it("uses the optional admin display name", () => {
    expect(eventTeamLabel("Manchester United", "Man Utd")).toBe("Man Utd");
    expect(eventTeamLabel("Manchester United", " MU ")).toBe("MU");
  });
  it("falls back to full names without generating initials", () => {
    expect(eventTeamLabel("Manchester United")).toBe("Manchester United");
    expect(eventTeamLabel("Manchester United", null)).toBe("Manchester United");
    expect(eventTeamLabel("Manchester United", "  ")).toBe("Manchester United");
    expect(eventTeamLabel("Team A")).toBe("Team A");
  });
});

it("scores unnamed goals and own goals for the correct side",()=>{
 expect(scoreEvents(fixture,[{...e("","goal",1),team_tag:"EDL001"},{...e("","own_goal",2),team_tag:"EDL001"}],rosters)).toEqual({home:1,away:1});
});
