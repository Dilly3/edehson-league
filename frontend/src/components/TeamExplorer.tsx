import { useEffect, useState } from "react";
import { Alert, Button, CircularProgress, Dialog, DialogActions, DialogContent, DialogTitle } from "@mui/material";
import { getPlayers, getPlayerStats, message } from "../api";
import TeamSummary from "./TeamSummary";
import type { Player, PlayerStats, Standing } from "../types";

type Selection = { teamTag: string; teamName: string } | { player: { id: string; name: string; jersey_number: string; team: string } };
export type ExplorerSelection = Selection;

export default function TeamExplorer({ selection, standings, seasonStats, onClose }: { selection: Selection; standings: Standing[] | null; seasonStats: PlayerStats[] | null; onClose: () => void }) {
  const initialPlayer = "player" in selection ? selection.player : null;
  const [player, setPlayer] = useState(initialPlayer);
  const [roster, setRoster] = useState<Player[]>([]);
  const [stats, setStats] = useState<PlayerStats[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [attempt, setAttempt] = useState(0);
  useEffect(() => {
    let active = true;
    setLoading(true);
    setError("");
    (async () => {
      try {
        // Fetch statistics when opening a player; a stats failure must not hide the roster.
        if (player) {
          const rows = await getPlayerStats();
          if (active) setStats(rows);
        } else if ("teamTag" in selection) {
          const rows = await getPlayers(selection.teamTag);
          if (active) setRoster(rows);
        }
      } catch (e) {
        if (active) setError(message(e));
      } finally {
        if (active) setLoading(false);
      }
    })();
    return () => { active = false; };
  }, [selection, player, attempt]);
  const current = player ? stats.find(s => s.player_id === player.id) : undefined;
  return (
    <Dialog open onClose={onClose} fullWidth maxWidth="sm">
      <DialogTitle>{player ? player.name : `${"teamName" in selection ? selection.teamName : "Team"} roster`}</DialogTitle>
      <DialogContent>
        {!player && "teamTag" in selection && <>
          <TeamSummary teamTag={selection.teamTag} standings={standings} stats={seasonStats} onPlayerClick={p => setPlayer({id: p.player_id, name: p.name, jersey_number: p.jersey_number, team: p.team_name})} />
          <h3 className="team-roster-heading">Roster</h3>
        </>}
        {loading ? <div role="status" aria-label="Loading player information"><CircularProgress size={24} /></div> : error ? (
          <Alert severity="error" action={<Button onClick={() => setAttempt(a => a + 1)}>Retry</Button>}>{error}</Alert>
        ) : player ? <>
          <p className="muted">{player.team} · Jersey #{player.jersey_number}</p>
          {current ? <>
            <p className="muted small">Season totals · Finalized matches only</p>
            <dl className="player-stat-cards">
              {([["Goals", current.goals], ["Yellow cards", current.yellow_cards], ["Red cards", current.red_cards], ["Own goals", current.own_goals]] as const).map(([label, value]) => (
                <div key={label}><dt>{label}</dt><dd>{value ?? "Unavailable"}</dd></div>
              ))}
            </dl>
            <p className="muted small">Own goals count towards the opposing team’s score.</p>
          </> : <Alert severity="info">No statistics are available for this player yet.</Alert>}
        </> : roster.length ? <div className="public-roster">
          {roster.map(p => <button key={p.id} className="public-roster-player" onClick={() => setPlayer({ id: p.id, name: p.name, jersey_number: p.jersey_number, team: "teamName" in selection ? selection.teamName : p.team })}>
            <span className="jersey">{p.jersey_number}</span>
            <span><strong>{p.name}</strong>{p.is_captain && <small>Captain</small>}</span>
            <span aria-hidden="true">›</span>
          </button>)}
        </div> : <p>No players have been registered for this team yet.</p>}
      </DialogContent>
      <DialogActions>
        {player && "teamTag" in selection && <Button onClick={() => setPlayer(null)}>Back to roster</Button>}
        <Button onClick={onClose}>Close</Button>
      </DialogActions>
    </Dialog>
  );
}
