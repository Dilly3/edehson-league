import type { PlayerStats, Standing } from "../types";
import { teamSeasonSummary } from "../football";

export default function TeamSummary({ teamTag, standings, stats, onPlayerClick }: {
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
      <div className="team-summary-scorers">
        <dt>{summary && summary.scorers.length > 1 ? "Joint top scorers" : "Top scorer"}</dt>
        <dd>{!summary ? "Unavailable" : summary.scorers.length ? <>
          {summary.scorers.map(player => <button className="text-link" key={player.player_id} onClick={() => onPlayerClick(player)}>{player.name}</button>)}
        </> : "No goals yet"}</dd>
        {summary && summary.goals > 0 && <small>{summary.goals} {summary.goals === 1 ? "goal" : "goals"}{summary.scorers.length > 1 ? " each" : ""}</small>}
      </div>
      <div><dt>Yellow cards</dt><dd>{summary?.yellowCards ?? "Unavailable"}</dd></div>
      <div><dt>Red cards</dt><dd>{summary?.redCards ?? "Unavailable"}</dd></div>
    </dl>
    <p>Full-time matches only · Own goals excluded from scorer totals.</p>
  </section>;
}
