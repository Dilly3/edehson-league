import type { PlayerStats, Standing } from "../types";
import { teamSeasonSummary } from "../football";

export default function TeamSummary({ teamTag, standings, stats }: {
  teamTag: string;
  standings: Standing[] | null;
  stats: PlayerStats[] | null;
  onPlayerClick: (player: PlayerStats) => void;
}) {
  const standing = standings?.find(row => row.team_tag === teamTag);
  const summary = stats === null ? null : teamSeasonSummary(teamTag, stats);
  return <section className="team-summary" aria-label="Team season summary">
    <dl className="team-summary-grid">
      <div>
        <dt>League position</dt>
        <dd>{standings === null ? "Unavailable" : standing ? `#${standing.position}` : "Not ranked"}</dd>
        {standing && <small>{standing.points} points · {standing.played} played</small>}
      </div>
      <div><dt>Goals scored</dt><dd>{standing?.goals_for ?? "Unavailable"}</dd></div>
      <div><dt>Goals conceded</dt><dd>{standing?.goals_against ?? "Unavailable"}</dd></div>
      <div><dt>Yellow cards</dt><dd>{summary?.yellowCards ?? "Unavailable"}</dd></div>
      <div><dt>Red cards</dt><dd>{summary?.redCards ?? "Unavailable"}</dd></div>
    </dl>
    <p>Full-time matches only.</p>
  </section>;
}
