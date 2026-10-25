import { Button } from "@mui/material";
import PlayArrowRounded from "@mui/icons-material/PlayArrowRounded";
import type { Fixture } from "../types";
import { hasPublishedScore, matchStatusLabel } from "../football";
import { Crest, SectionHeading } from "./Shared";

export default function LiveBroadcastSection({ fixtures, unavailable, onWatch }: {
  fixtures: Fixture[];
  unavailable: boolean;
  onWatch: (fixture: Fixture) => void;
}) {
  const broadcasts = fixtures.filter(f => f.broadcast_url && ["scheduled", "live", "half_time"].includes(f.status));
  return <section className="panel broadcast-section" aria-label="Live Broadcast">
    <SectionHeading title="Live Broadcast" />
    {unavailable ? <p className="broadcast-empty">Broadcast information is currently unavailable. Refresh to try again.</p> : broadcasts.length === 0 ?
      <p className="broadcast-empty">No match broadcast is available right now. Check back on matchday.</p> : <>
      <div className="broadcast-cards">{broadcasts.map(f => <article className="broadcast-card" key={f.id}>
        <div className="broadcast-card-top"><span className="eyebrow">Matchday {f.matchday}</span><span>{matchStatusLabel(f.status)}</span></div>
        <div className="broadcast-teams">
          <div><Crest tag={f.home_team_tag} name={f.home_team} size={44} /><strong>{f.home_team}</strong></div>
          <span className="broadcast-score">{hasPublishedScore(f) ? `${f.home_score} : ${f.away_score}` : "vs"}</span>
          <div><Crest tag={f.away_team_tag} name={f.away_team} size={44} /><strong>{f.away_team}</strong></div>
        </div>
        <p className="muted">{f.venue || "Venue to be confirmed"}</p>
        <Button variant="contained" startIcon={<PlayArrowRounded />} onClick={() => onWatch(f)} aria-label={`Watch ${f.home_team} vs ${f.away_team}`}>Watch broadcast</Button>
      </article>)}</div>
      <p className="broadcast-note muted">Playback is available while the broadcaster is streaming.</p>
    </>}
  </section>;
}
