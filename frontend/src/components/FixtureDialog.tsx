import { useEffect, useState } from "react";
import {
  Alert,
  Button,
  CircularProgress,
  Checkbox,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  FormControlLabel,
  IconButton,
  MenuItem,
  Tab,
  Tabs,
  TextField,
} from "@mui/material";
import AddRounded from "@mui/icons-material/AddRounded";
import DeleteOutlineRounded from "@mui/icons-material/DeleteOutlineRounded";
import ArrowUpwardRounded from "@mui/icons-material/ArrowUpwardRounded";
import ArrowDownwardRounded from "@mui/icons-material/ArrowDownwardRounded";
import { api, ApiError, getPlayers, message, updateFixtureVenue } from "../api";
import {
  eventLabels,
  eventTeamLabel,
  matchStatusLabel,
  formatKickoff,
  kickoffFromInput,
  kickoffToInput,
  scoreEvents,
  validateEvents,
  venueFromInput,
} from "../football";
import type {
  EventType,
  Fixture,
  Team,
  FixtureDetail,
  MatchEvent,
  Player,
  PlayerStats,
  Standing,
} from "../types";
import LiveBroadcast from "./LiveBroadcast";
import MatchPreview from "./MatchPreview";
import { Crest } from "./Shared";
export default function FixtureDialog({
  fixture,
  teams,
  initialWatch = false,
  standings,
  seasonStats,
  onPlayerClick,
  token,
  onClose,
  onSaved,
}: {
  fixture: Fixture;
  teams: Team[];
  initialWatch?: boolean;
  standings: Standing[] | null;
  seasonStats: PlayerStats[] | null;
  onPlayerClick: (player: PlayerStats) => void;
  token: string;
  onClose: () => void;
  onSaved: () => void;
}) {
  const [detail, setDetail] = useState<FixtureDetail | null>(null),
    [rosters, setRosters] = useState<Record<string, Player[]>>({}),
    [events, setEvents] = useState<MatchEvent[]>([]),
    [score, setScore] = useState({ home: 0, away: 0 }),
    [publishStatus, setPublishStatus] = useState("live"),
    [kickoff, setKickoff] = useState(""),
    [moveMatchday, setMoveMatchday] = useState(true),
    [broadcast, setBroadcast] = useState(false),
    [venue, setVenue] = useState(""),
    [tab, setTab] = useState(initialWatch ? 5 : 0),
    [loading, setLoading] = useState(true),
    [busy, setBusy] = useState(false),
    [error, setError] = useState(""),
    [conflict, setConflict] = useState(false),
    [rosterError, setRosterError] = useState("");
  const load = async () => {
    setLoading(true);
    setError("");
    setRosterError("");
    try {
      const d = await api<FixtureDetail>(`/fixtures/${fixture.id}`);
      setDetail(d);
      setPublishStatus(["live", "half_time", "finalized"].includes(d.fixture.status) ? d.fixture.status : "live");
      setEvents(d.events ?? []);
      setScore({
        home: d.fixture.home_score ?? 0,
        away: d.fixture.away_score ?? 0,
      });
      setKickoff(kickoffToInput(d.fixture.kickoff_at));
      setVenue(d.fixture.venue ?? "");
      setBroadcast(!!d.fixture.broadcast_url);
      setConflict(false);
      const r = await Promise.allSettled([
        getPlayers(d.fixture.home_team_tag),
        getPlayers(d.fixture.away_team_tag),
      ]);
      if (r[0].status === "fulfilled" && r[1].status === "fulfilled")
        setRosters({
          [d.fixture.home_team_tag]: r[0].value,
          [d.fixture.away_team_tag]: r[1].value,
        });
      else
        setRosterError(
          "Could not load both rosters. Reload before editing results.",
        );
    } catch (e) {
      setError(message(e));
      setDetail(null);
    } finally {
      setLoading(false);
    }
  };
  useEffect(() => {
    void load();
  }, [fixture.id]);
  useEffect(() => {
    if (token) return;
    let active = true;
    const timer = window.setInterval(() => {
      if (document.visibilityState !== "visible") return;
      void api<FixtureDetail>(`/fixtures/${fixture.id}`).then(d => { if (active) setDetail(d); }).catch(() => {});
    }, 30000);
    return () => { active = false; window.clearInterval(timer); };
  }, [fixture.id, token, tab]);
  const current = detail?.fixture ?? fixture;
  const calculated = scoreEvents(current, events, rosters);
  const validation = validateEvents(events);
  const mismatch =
    calculated.home !== score.home || calculated.away !== score.away;
  const available = Object.values(rosters).flat();
  const patchEvent = (index: number, patch: Partial<MatchEvent>) =>
    setEvents(events.map((e, i) => (i === index ? { ...e, ...patch } : e)));
  const moveEvent = (index: number, direction: number) => {
    const next = [...events];
    [next[index], next[index + direction]] = [
      next[index + direction],
      next[index],
    ];
    setEvents(next);
  };
  const save = async (kind: "result" | "kickoff" | "venue" | "broadcast") => {
    setError("");
    if (kind === "result" && (validation || mismatch)) {
      setError(validation ?? "The goal events must match the final score.");
      return;
    }
    if (kind === "venue") {
      const normalized = venueFromInput(venue);
      if (
        normalized !== null &&
        (Array.from(normalized).length < 2 ||
          Array.from(normalized).length > 120)
      ) {
        setError(
          "Venue must contain 2–120 characters. Leave it empty to clear it.",
        );
        return;
      }
    }
    setBusy(true);
    try {
      if (kind === "broadcast")
        await api(`/fixtures/${fixture.id}/broadcast`, { method: "PATCH", body: JSON.stringify({ broadcast_url: broadcast ? "https://d3529bfd1f67.eu-west-1.playback.live-video.net/api/video/v1/eu-west-1.821643761950.channel.T8wnLDiQ8H3y.m3u8" : null }) }, token);
      else if (kind === "venue")
        await updateFixtureVenue(fixture.id, venueFromInput(venue), token);
      else if (kind === "kickoff")
        await api(
          `/fixtures/${fixture.id}/kickoff`,
          {
            method: "PATCH",
            body: JSON.stringify({ kickoff_at: kickoffFromInput(kickoff), move_matchday: moveMatchday }),
          },
          token,
        );
      else
        await api(
          `/fixtures/${fixture.id}/result`,
          {
            method: "PUT",
            body: JSON.stringify({
              status: publishStatus,
              home_score: score.home,
              away_score: score.away,
              revision: current.revision,
              events: events.map((e) => ({
                player_id: e.player_id,
                team_tag: e.player_id?Object.entries(rosters).find(([,players])=>players.some(p=>p.id===e.player_id))?.[0]:e.team_tag,
                type: e.type,
                minute: e.minute,
              })),
            }),
          },
          token,
        );
      onSaved();
      onClose();
    } catch (e) {
      setError(message(e));
      if (e instanceof ApiError && e.status === 409 && kind === "result")
        setConflict(true);
    } finally {
      setBusy(false);
    }
  };
  return (
    <Dialog className="screen-fit-dialog" open onClose={busy ? undefined : onClose} fullWidth maxWidth="md">
      <DialogTitle>
        <span className="eyebrow">Matchday {current.matchday}</span>
        <div className="match-dialog-title">
          {current.home_team} vs {current.away_team}
        </div>
      </DialogTitle>
      <DialogContent>
        <div className="dialog-form">
          {loading ? (
            <div className="loading">
              <CircularProgress size={28} />
              <span>Loading match details…</span>
            </div>
          ) : (
            <>
              {error && (
                <Alert
                  severity="error"
                  action={
                    (tab === 1 && conflict) || !detail ? (
                      <Button color="inherit" onClick={() => void load()}>
                        Reload
                      </Button>
                    ) : undefined
                  }
                >
                  {error}
                  {conflict && " Reloading discards your unsaved edits."}
                </Alert>
              )}
              {detail && (
                <>
                  <div className="match-scoreboard">
                    <div>
                      <Crest
                        tag={current.home_team_tag}
                        name={current.home_team}
                        size={54}
                      />
                      <strong>{current.home_team}</strong>
                      <small>Home</small>
                    </div>
                    <div className="big-score">
                      {current.home_score ?? "–"} <span>:</span>{" "}
                      {current.away_score ?? "–"}
                    </div>
                    <div>
                      <Crest
                        tag={current.away_team_tag}
                        name={current.away_team}
                        size={54}
                      />
                      <strong>{current.away_team}</strong>
                      <small>Away</small>
                    </div>
                  </div>
                  <div className="match-meta">
                    <span>
                      {matchStatusLabel(current.status)} ·{" "}
                      {formatKickoff(current.kickoff_at)} · WAT
                    </span>
                    {token && (
                      <Button
                        size="small"
                        variant="outlined"
                        onClick={() => setTab(2)}
                        disabled={
                          busy ||
                          ["live", "half_time", "finalized"].includes(
                            current.status,
                          )
                        }
                        title={
                          ["live", "half_time", "finalized"].includes(
                            current.status,
                          )
                            ? "Kickoff cannot be changed after a match starts"
                            : undefined
                        }
                      >
                        Edit kickoff
                      </Button>
                    )}
                  </div>
                  <p className="match-venue">
                    {current.venue || "Venue to be confirmed"}
                  </p>
                  <Tabs
                    variant="scrollable"
                    scrollButtons="auto"
                    value={tab}
                    onChange={(_, v) => setTab(v)}
                    aria-label="Match detail sections"
                  >
                    <Tab label="Match events" />
                    {token && (
                      <Tab
                        label={
                          current.status === "finalized"
                            ? "Correct result"
                            : "Update match"
                        }
                      />
                    )}{" "}
                    {token && (
                      <Tab
                        label="Edit kickoff"
                        disabled={["live", "half_time", "finalized"].includes(current.status)}
                      />
                    )}
                    {token && <Tab label="Venue" />}
                    {token && <Tab label="Broadcast" value={4} />}
                    {current.broadcast_url && <Tab label="Watch live" value={5} />}
                  </Tabs>
                  {tab === 0 && (
                    <div>
                      {!detail.events.length ? (
                        <MatchPreview fixture={current} standings={standings} stats={seasonStats} onPlayerClick={onPlayerClick} />
                      ) : (
                        <div className="timeline team-timeline">
                          <div className="timeline-teams" aria-hidden="true">
                            <strong>{current.home_team}</strong>
                            <span />
                            <strong>{current.away_team}</strong>
                          </div>
                          {detail.events.map((e, i) => {
                            const teamTag = e.team_tag || Object.entries(rosters).find(([, players]) => players.some(p => p.id === e.player_id))?.[0];
                            const side = teamTag === current.home_team_tag ? "home" : teamTag === current.away_team_tag ? "away" : "unknown";
                            const teamName = side === "home" ? current.home_team : side === "away" ? current.away_team : "Team unavailable";
                            return (
                              <div className={`timeline-row timeline-${side}`} key={e.id ?? i}>
                                <span className="event-minute">
                                  {e.minute === null ? "—" : `${e.minute}′`}
                                </span>
                                <div className="timeline-event">
                                  <span className={`event-dot ${e.type}`} aria-hidden="true" />
                                  <div className="timeline-event-copy">
                                    <strong>{e.player_name || "Scorer not recorded"}</strong>
                                    <small>{eventLabels[e.type] ?? "Unsupported event"} · <span title={teamName}>{side === "unknown" ? teamName : eventTeamLabel(teamName, teams.find(t => t.tag === teamTag)?.display_name)}</span></small>
                                    {e.type === "own_goal" && side !== "unknown" && <small>Counts for <span title={side === "home" ? current.away_team : current.home_team}>{eventTeamLabel(side === "home" ? current.away_team : current.home_team, teams.find(t => t.tag === (side === "home" ? current.away_team_tag : current.home_team_tag))?.display_name)}</span></small>}
                                  </div>
                                </div>
                              </div>
                            );
                          })}
                        </div>
                      )}
                    </div>
                  )}
                  {token && tab === 1 && (
                    <>
                      <Alert severity="info">
                        Enter every goal and card in match order. Scorers are optional; cards require a player. Select the responsible team for unnamed goals. Own goals
                        count for the opposing team. Saving replaces the
                        current score and event list and publishes immediately.
                        Standings and player totals update only at full time.
                      </Alert>
                      {rosterError && (
                        <Alert
                          severity="error"
                          action={
                            <Button onClick={() => void load()}>Reload</Button>
                          }
                        >
                          {rosterError}
                        </Alert>
                      )}
                      <TextField select label="Match status" value={publishStatus} onChange={(e) => setPublishStatus(e.target.value)} disabled={busy || current.status === "finalized"} fullWidth margin="normal"
                        helperText={current.status === "finalized" ? "Full-time results can be corrected but cannot return to live." : "Choose Live for either half, Half-time at the break, or Full-time when the match ends."}>
                        <MenuItem value="live">Live</MenuItem>
                        <MenuItem value="half_time">Half-time</MenuItem>
                        <MenuItem value="finalized">Full-time</MenuItem>
                      </TextField>
                      <div className="score-inputs">
                        <TextField
                          label={`${current.home_team} goals`}
                          type="number"
                          value={score.home}
                          onChange={(e) =>
                            setScore({
                              ...score,
                              home:
                                e.target.value === ""
                                  ? 0
                                  : Number(e.target.value),
                            })
                          }
                          slotProps={{ htmlInput: { min: 0, step: 1 } }}
                        />
                        <span>–</span>
                        <TextField
                          label={`${current.away_team} goals`}
                          type="number"
                          value={score.away}
                          onChange={(e) =>
                            setScore({
                              ...score,
                              away:
                                e.target.value === ""
                                  ? 0
                                  : Number(e.target.value),
                            })
                          }
                          slotProps={{ htmlInput: { min: 0, step: 1 } }}
                        />
                      </div>
                      <div className="section-heading">
                        <h3>Goals & cards</h3>
                        <Button
                          startIcon={<AddRounded />}
                          onClick={() =>
                            setEvents([
                              ...events,
                              { player_id: "", type: "goal", minute: null },
                            ])
                          }
                          disabled={!!rosterError || events.length >= 500}
                        >
                          Add event
                        </Button>
                      </div>
                      {!events.length && (
                        <p className="muted">
                          No events added. A 0–0 result can be saved without
                          events.
                        </p>
                      )}
                      {events.map((e, i) => (
                        <div className={`event-editor${!e.player_id&&(e.type==="goal"||e.type==="own_goal")?" event-editor-unnamed":""}`} key={i}>
                          <span className="event-order">{i + 1}</span>
                          <TextField
                            select
                            label={e.type==="goal"||e.type==="own_goal"?"Scorer (optional)":"Player (required)"}
                            value={e.player_id}
                            onChange={(ev) =>
                              patchEvent(i, { player_id: ev.target.value, team_tag: ev.target.value?Object.entries(rosters).find(([,players])=>players.some(p=>p.id===ev.target.value))?.[0]:e.team_tag })
                            }
                          >
                            <MenuItem value="">
                              {e.type==="goal"||e.type==="own_goal"?"Scorer not recorded":"Select a player"}
                            </MenuItem>
                            {available.map((p) => (
                              <MenuItem key={p.id} value={p.id}>
                                {p.name} · #{p.jersey_number} ·{" "}
                                {(rosters[current.home_team_tag] ?? []).some(
                                  (x) => x.id === p.id,
                                )
                                  ? current.home_team
                                  : current.away_team}
                              </MenuItem>
                            ))}
                          </TextField>
                          {!e.player_id&&(e.type==="goal"||e.type==="own_goal")&&<TextField select label={e.type==="own_goal"?"Team conceding own goal":"Scoring team"} value={e.team_tag??""} onChange={ev=>patchEvent(i,{team_tag:ev.target.value})} required><MenuItem value="" disabled>Select a team</MenuItem><MenuItem value={current.home_team_tag}>{current.home_team}</MenuItem><MenuItem value={current.away_team_tag}>{current.away_team}</MenuItem></TextField>}
                          <TextField
                            select
                            label="Event"
                            value={e.type}
                            onChange={(ev) =>
                              patchEvent(i, {
                                type: ev.target.value as EventType,
                              })
                            }
                          >
                            {Object.entries(eventLabels).map(
                              ([value, label]) => (
                                <MenuItem key={value} value={value}>
                                  {label}
                                </MenuItem>
                              ),
                            )}
                          </TextField>
                          <TextField
                            label="Minute"
                            placeholder="Unknown"
                            type="number"
                            value={e.minute ?? ""}
                            onChange={(ev) =>
                              patchEvent(i, {
                                minute:
                                  ev.target.value === ""
                                    ? null
                                    : Number(ev.target.value),
                              })
                            }
                            slotProps={{ htmlInput: { min: 0, step: 1 } }}
                          />
                          <div className="event-actions">
                            <IconButton
                              size="small"
                              aria-label={`Move event ${i + 1} up`}
                              disabled={i === 0}
                              onClick={() => moveEvent(i, -1)}
                            >
                              <ArrowUpwardRounded fontSize="small" />
                            </IconButton>
                            <IconButton
                              size="small"
                              aria-label={`Move event ${i + 1} down`}
                              disabled={i === events.length - 1}
                              onClick={() => moveEvent(i, 1)}
                            >
                              <ArrowDownwardRounded fontSize="small" />
                            </IconButton>
                            <IconButton
                              size="small"
                              aria-label={`Remove event ${i + 1}`}
                              onClick={() =>
                                setEvents(events.filter((_, j) => j !== i))
                              }
                            >
                              <DeleteOutlineRounded fontSize="small" />
                            </IconButton>
                          </div>
                        </div>
                      ))}
                      <Alert
                        severity={
                          validation || mismatch ? "warning" : "success"
                        }
                      >
                        {validation ??
                          `Goal events produce ${calculated.home}–${calculated.away}.${mismatch ? " This must match your current score." : " Score and goal events match."}`}
                      </Alert>
                    </>
                  )}
                  {tab === 5 && current.broadcast_url && <LiveBroadcast url={current.broadcast_url} />}
                  {token && tab === 4 && <TextField select label="Match broadcast" value={broadcast ? "attached" : "none"} onChange={e => setBroadcast(e.target.value === "attached")} helperText="Attach our league channel only to the match being broadcast. Remove it here before attaching it to another fixture. This does not start or stop Larix."><MenuItem value="none">No broadcast</MenuItem><MenuItem value="attached">League live channel</MenuItem></TextField>}
                  {token && tab === 3 && (
                    <>
                      <TextField
                        label="Match venue"
                        value={venue}
                        onChange={(e) => setVenue(e.target.value)}
                        placeholder="Edehson Community Ground"
                        disabled={busy}
                        slotProps={{ htmlInput: { maxLength: 120 } }}
                        helperText="2–120 characters. Leave empty to mark the venue as unconfirmed."
                      />
                      <p className="muted">
                        Venue corrections are allowed before or after the match.
                        Kickoff time is saved separately.
                      </p>
                    </>
                  )}
                  {token && tab === 2 && !["live", "half_time", "finalized"].includes(current.status) && (
                    <>
                      <TextField
                        label="Kickoff date & time (Nigeria / WAT)"
                        type="datetime-local"
                        value={kickoff}
                        onChange={(e) => setKickoff(e.target.value)}
                        slotProps={{ inputLabel: { shrink: true } }}
                      />
                      <FormControlLabel
                        control={<Checkbox checked={moveMatchday} onChange={(e) => setMoveMatchday(e.target.checked)} />}
                        label={`Apply to the whole of matchday ${current.matchday}`}
                      />
                      <p className="muted">
                        All kickoff times use Nigeria / WAT (UTC+01:00). Games
                        in one matchday must share a date, and a date cannot be
                        shared by different matchdays. With the box ticked, the
                        other unplayed games in this matchday move to the same
                        date and keep their own kickoff times. Leave the field
                        empty to clear the kickoff (and the whole matchday's
                        unplayed dates when ticked).
                      </p>
                    </>
                  )}
                </>
              )}
            </>
          )}
        </div>
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose} disabled={busy}>
          Close
        </Button>
        {token && detail && !loading && tab > 0 && tab < 5 && (
          <Button
            variant="contained"
            onClick={() =>
              void save(tab === 1 ? "result" : tab === 2 ? "kickoff" : tab === 4 ? "broadcast" : "venue")
            }
            disabled={
              busy ||
              (tab === 1 && conflict) ||
              (tab === 1 &&
                (!!rosterError ||
                  !!validation ||
                  mismatch ||
                  !Number.isInteger(score.home) ||
                  !Number.isInteger(score.away) ||
                  score.home < 0 ||
                  score.away < 0 ||
                  !["scheduled", "live", "half_time", "finalized"].includes(current.status)))
            }
          >
            {busy
              ? "Saving…"
              : tab === 1
                ? (publishStatus === "finalized" ? "Publish full-time result" : `Publish ${matchStatusLabel(publishStatus).toLowerCase()} update`)
                : tab === 2
                  ? "Save kickoff"
                  : tab === 4 ? "Save broadcast" : "Save venue"}
          </Button>
        )}
      </DialogActions>
    </Dialog>
  );
}
