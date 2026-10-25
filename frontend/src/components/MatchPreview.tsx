import type { Fixture, PlayerStats, Standing } from "../types";
import { teamSeasonSummary, defenceSummary } from "../football";
import { Crest } from "./Shared";

export default function MatchPreview({ fixture, standings, stats, onPlayerClick }: {
  fixture: Fixture;
  standings: Standing[] | null;
  stats: PlayerStats[] | null;
  onPlayerClick: (player: PlayerStats) => void;
}) {
  const title = fixture.status === "finalized" ? "Team snapshot" : ["live", "half_time"].includes(fixture.status) ? "Team form" : "Match preview";
  return <section className="match-preview" aria-label={title}>
    <h3>{title}</h3>
    <p className="match-preview-note">No goals or cards have been recorded for this fixture. Here is how both teams have performed in the league.</p>
    <div className="match-preview-grid">
      {[{tag: fixture.home_team_tag, name: fixture.home_team, label: "Home"}, {tag: fixture.away_team_tag, name: fixture.away_team, label: "Away"}].map(team => {
        const standing = standings?.find(row => row.team_tag === team.tag);
        const summary = stats === null ? null : teamSeasonSummary(team.tag, stats);
        return <article key={team.tag} className="match-preview-team">
          <header><Crest tag={team.tag} name={team.name} size={38} /><div><h4>{team.name}</h4><span>{team.label}{standing ? ` · #${standing.position} in the league` : ""}</span></div></header>
          <dl className="match-preview-numbers">
            <div><dt>Goals scored</dt><dd>{standing?.goals_for ?? "—"}</dd></div>
            <div><dt>Goals conceded</dt><dd>{standing?.goals_against ?? "—"}</dd></div>
          </dl>
          <h5>Attack</h5>
          <p>{!standing ? "Team scoring statistics are currently unavailable." : standing.played === 0 ? "This team has no matches played yet. Their attacking record is still to be established." : `${team.name} have scored ${standing.goals_for} ${standing.goals_for === 1 ? "goal" : "goals"} in ${standing.played} ${standing.played === 1 ? "match" : "matches"} played.`}</p>
          <h5>{summary && summary.scorers.length > 1 ? "Players to watch" : "Player to watch"}</h5>
          {!summary ? <p>Scorer statistics are currently unavailable.</p> : summary.scorers.length ? <>
            <ul className="match-preview-scorers">{summary.scorers.map(player => <li key={player.player_id}><button type="button" className="text-link" onClick={() => onPlayerClick(player)}>{player.name}</button><span>{player.goals} {player.goals === 1 ? "goal" : "goals"}</span></li>)}</ul>
            <p className="match-preview-note">{summary.scorers.length > 1 ? "Joint leading scorers for their team." : "Their leading league scorer."}</p>
          </> : <p>No league goalscorer has emerged yet.</p>}
          <h5>Defence</h5>
          <p>{standing ? defenceSummary(standing.played, standing.goals_against) : "Defensive statistics are currently unavailable."}</p>
        </article>;
      })}
    </div>
    <p className="match-preview-note">Team goals include own goals; individual scorer totals exclude them. These are season records, not predictions.</p>
  </section>;
}
