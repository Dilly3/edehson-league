export interface Team {
  display_name?: string | null;
  id: string;
  name: string;
  tag: string;
}
export interface Player {
  id: string;
  name: string;
  jersey_number: string;
  team: string;
  team_tag: string;
  player_tag: string;
  is_captain: boolean;
}
export interface Fixture {
  id: string;
  home_team_tag: string;
  home_team: string;
  away_team_tag: string;
  away_team: string;
  matchday: number;
  kickoff_at: string | null;
  venue: string | null;
  broadcast_url?: string | null;
  home_score: number | null;
  away_score: number | null;
  status: string;
  revision: number;
}
export interface Standing {
  position: number;
  team_tag: string;
  team_name: string;
  played: number;
  won: number;
  drawn: number;
  lost: number;
  goals_for: number;
  goals_against: number;
  goal_difference: number;
  points: number;
}
export interface PlayerStats {
  player_id: string;
  player_tag: string;
  name: string;
  team_tag: string;
  team_name: string;
  jersey_number: string;
  goals: number;
  yellow_cards: number;
  red_cards: number;
  own_goals?: number;
}
export type EventType = "goal" | "own_goal" | "yellow_card" | "red_card";
export interface MatchEvent {
  id?: string;
  player_id: string;
  player_name?: string;
  team_tag?: string;
  type: EventType;
  minute: number | null;
  position?: number;
}
export interface FixtureDetail {
  fixture: Fixture;
  events: MatchEvent[];
}
export interface ResultInput {
  status?: "live" | "half_time" | "finalized";
  home_score: number;
  away_score: number;
  revision: number;
  events: MatchEvent[];
}
export interface LeagueData {
  teams: Team[];
  fixtures: Fixture[];
  standings: Standing[];
  stats: PlayerStats[];
}
