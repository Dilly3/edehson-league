import ScrollTable from "./components/ScrollTable";
import LeagueLeaders from "./components/LeagueLeaders";
import { useCallback, useEffect, useRef, useState } from "react";
import {
  Alert,
  Button,
  CircularProgress,
  IconButton,
  MenuItem,
  Select,
  Snackbar,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableRow,
  TextField,
  Tooltip,
} from "@mui/material";
import DarkModeOutlined from "@mui/icons-material/DarkModeOutlined";
import LightModeOutlined from "@mui/icons-material/LightModeOutlined";
import { useAppearance } from "./theme";
import TeamRenameDialog from "./components/TeamRenameDialog";
import TeamSummary from "./components/TeamSummary";
import TeamExplorer, { type ExplorerSelection } from "./components/TeamExplorer";
import ScrollNavigation from "./components/ScrollNavigation";
import ArrowForwardRounded from "@mui/icons-material/ArrowForwardRounded";
import RefreshRounded from "@mui/icons-material/RefreshRounded";
import AddRounded from "@mui/icons-material/AddRounded";
import LockOutlined from "@mui/icons-material/LockOutlined";
import LogoutRounded from "@mui/icons-material/LogoutRounded";
import EditOutlined from "@mui/icons-material/EditOutlined";
import DeleteOutlineRounded from "@mui/icons-material/DeleteOutlineRounded";
import MilitaryTechOutlined from "@mui/icons-material/MilitaryTechOutlined";
import PlayerManagementDialog, {
  type PlayerAction,
} from "./components/PlayerManagementDialog";
import PlaceOutlined from "@mui/icons-material/PlaceOutlined";
import SearchRounded from "@mui/icons-material/SearchRounded";
import { api, getPlayers, getPlayerStats, message } from "./api";
import type { Fixture, LeagueData, Player, PlayerStats } from "./types";
import { kickoffParts, awaitingResult, scheduleSummary, matchStatusLabel, hasPublishedScore } from "./football";
import {
  Crest,
  Empty,
  LeagueTable,
  MoreButton,
  SectionHeading,
} from "./components/Shared";
import AdminForms, { type AdminAction } from "./components/AdminForms";
import MembershipPage from "./components/MembershipPage";
import { ViewerPINSettings, logoutViewer } from "./components/ViewerAccess";
import { setAdminCredential } from "./api";
import AdminLogin from "./components/AdminLogin";
import { isAdminPath, resolvePage, pageURL, type Page } from "./navigation";
import LiveBroadcastSection from "./components/LiveBroadcastSection";
import FixtureDialog from "./components/FixtureDialog";

const pages: { id: Page; label: string }[] = [
  { id: "overview", label: "Overview" },
  { id: "standings", label: "Table" },
  { id: "fixtures", label: "Fixtures" },
  { id: "players", label: "Players" },
  { id: "teams", label: "Teams" },
  { id: "membership", label: "Roaster" },
];
const pageText: Record<Exclude<Page, "overview">, { title: string; note: string }> = {
  standings: {
    title: "League table",
    note: "Ranked by points, then goal difference, then goals scored.",
  },
  fixtures: {
    title: "Fixtures & results",
    note: "Select a match for its goals, cards and details. Times are WAT.",
  },
  players: {
    title: "Player stats",
    note: "Yellow and red cards from finalized matches.",
  },
  teams: {
    title: "Teams",
    note: "Pick a team to see its squad. Select a player for season totals.",
  },
  membership: {
    title: "Roaster",
    note: "Member register, monthly dues, card fines and payment history.",
  },
  admin: {
    title: "Administration",
    note: "Teams, rosters, the season schedule and match results.",
  },
};
const emptyData: LeagueData = {
  teams: [],
  fixtures: [],
  standings: [],
  stats: [],
};
function initialPage(): Page {
  return resolvePage(window.location.pathname, window.location.hash);
}
export default function App() {
  const { mode, toggle } = useAppearance();
  const adminArea = isAdminPath(window.location.pathname);
  const [page, setPage] = useState<Page>(initialPage),
    [data, setData] = useState<LeagueData>(emptyData),
    [loading, setLoading] = useState(true),
    [refreshing, setRefreshing] = useState(false),
    [errors, setErrors] = useState<string[]>([]),
    [updated, setUpdated] = useState<Date | null>(null),
    [token, setToken] = useState(""),
    [action, setAction] = useState<AdminAction>(null),
    [selectedFixture, setSelectedFixture] = useState<Fixture | null>(null),
    [watchFixtureID, setWatchFixtureID] = useState(""),
    [toast, setToast] = useState(""),
    [round, setRound] = useState("all"),
    [status, setStatus] = useState("all"),
    [search, setSearch] = useState(""),
    [selectedTeam, setSelectedTeam] = useState(""),
    [renamingTeam, setRenamingTeam] = useState(false),
    [roster, setRoster] = useState<Player[]>([]),
    [playerAction, setPlayerAction] = useState<{
      player: Player;
      teamTag: string;
      action: PlayerAction;
    } | null>(null),
    [rosterLoading, setRosterLoading] = useState(false),
    [rosterError, setRosterError] = useState("");
  const [now, setNow] = useState(Date.now);
  useEffect(() => {
    const timer = window.setInterval(() => setNow(Date.now()), 60000);
    return () => window.clearInterval(timer);
  }, []);
  const [explorer, setExplorer] = useState<ExplorerSelection | null>(null);
  const openTeam = (teamTag: string, teamName: string) => setExplorer({ teamTag, teamName });
  const generation = useRef(0),
    rosterGeneration = useRef(0);
  const refresh = useCallback(async () => {
    if(adminArea&&!token)return;
    const g = ++generation.current;
    setRefreshing(true);
    const requests = await Promise.allSettled([
      api<LeagueData["teams"]>("/teams"),
      api<LeagueData["fixtures"]>("/fixtures"),
      api<LeagueData["standings"]>("/standings"),
      getPlayerStats(),
    ]);
    if (g !== generation.current) return;
    const keys = ["teams", "fixtures", "standings", "stats"] as const;
    const names = ["Teams", "Fixtures", "Standings", "Player statistics"];
    setData((previous) => {
      const next = { ...previous };
      requests.forEach((r, i) => {
        if (r.status === "fulfilled") {
          Object.assign(next, { [keys[i]]: r.value ?? [] });
        }
      });
      return next;
    });
    // Compute errors independently from React's state updater lifecycle.
    const currentErrors = requests.flatMap((r, i) =>
      r.status === "rejected" ? [`${names[i]}: ${message(r.reason)}`] : [],
    );
    setErrors(currentErrors);
    if (currentErrors.length === 0) setUpdated(new Date());
    setLoading(false);
    setRefreshing(false);
  }, [adminArea,token]);
  useEffect(() => {
    void refresh();
  }, [refresh]);
  useEffect(() => {
    if (token) return;
    const timer = window.setInterval(() => {
      if (document.visibilityState === "visible") void refresh();
    }, 30000);
    return () => window.clearInterval(timer);
  }, [refresh, token]);
  const navigate = (next: Page) => {
    window.history.pushState(null, "", pageURL(next, adminArea));
    if (!isAdminPath(window.location.pathname)) setToken("");
    setPage(next);
    setSearch("");
  };
  useEffect(() => {
    const update = () => {
      setPage(initialPage());
      if (!isAdminPath(window.location.pathname)) setToken("");
      setSearch("");
    };
    window.addEventListener("hashchange", update);
    window.addEventListener("popstate", update);
    return () => { window.removeEventListener("hashchange", update); window.removeEventListener("popstate", update); };
  }, []);
  useEffect(() => {
    if (!selectedTeam && data.teams.length) setSelectedTeam(data.teams[0].tag);
  }, [data.teams, selectedTeam]);
  const loadRoster = useCallback(async () => {
    if (!selectedTeam || (adminArea&&!token)) return;
    const g = ++rosterGeneration.current;
    setRosterLoading(true);
    setRosterError("");
    try {
      const rows = await getPlayers(selectedTeam);
      if (g === rosterGeneration.current) setRoster(rows);
    } catch (e) {
      if (g === rosterGeneration.current) {
        setRosterError(message(e));
        setRoster([]);
      }
    } finally {
      if (g === rosterGeneration.current) setRosterLoading(false);
    }
  }, [selectedTeam,adminArea,token]);
  useEffect(() => {
    void loadRoster();
  }, [loadRoster]);
  const saved = () => {
    setToast("Changes saved to the league.");
    void refresh();
    void loadRoster();
  };
  const adminAction = (next: AdminAction) => {
    if (!token || next === "key") { navigate("admin"); return; }
    setAction(next);
  };
  const unavailable = (resource: string) =>
    errors.some((e) => e.startsWith(resource + ":"));
  const completed = data.fixtures.filter((f) => f.status === "finalized");
  const upcoming = data.fixtures
    .filter((f) => f.status === "scheduled" && !awaitingResult(f, now))
    .sort(
      (a, b) =>
        (a.kickoff_at ? new Date(a.kickoff_at).getTime() : Infinity) -
          (b.kickoff_at ? new Date(b.kickoff_at).getTime() : Infinity) ||
        a.matchday - b.matchday ||
        a.home_team.localeCompare(b.home_team),
    );
  const latestResults = [...completed].sort(
    (a, b) =>
      b.matchday - a.matchday ||
      (b.kickoff_at ? new Date(b.kickoff_at).getTime() : 0) -
        (a.kickoff_at ? new Date(a.kickoff_at).getTime() : 0),
  );
  const pendingResults = data.fixtures.filter(f => awaitingResult(f, now));
  const schedule = scheduleSummary(data.fixtures, data.teams.length);
  const next = upcoming[0];
  const inProgress = data.fixtures.filter(f => ["live", "half_time"].includes(f.status));
  const featured = inProgress[0] ?? next ?? latestResults[0] ?? pendingResults[0];
  const totalGoals = completed.reduce(
    (sum, f) => sum + (f.home_score ?? 0) + (f.away_score ?? 0),
    0,
  );
  const seasonTotals = (
    <div className="season-totals">
      <dl className="figures table-figures" aria-label="League season totals">
        <Stat label="Goals scored" value={unavailable("Fixtures") && !data.fixtures.length ? null : totalGoals} />
        <Stat label="Yellow cards" value={unavailable("Player statistics") && !data.stats.length ? null : data.stats.reduce((sum, p) => sum + p.yellow_cards, 0)} />
        <Stat label="Red cards" value={unavailable("Player statistics") && !data.stats.length ? null : data.stats.reduce((sum, p) => sum + p.red_cards, 0)} />
      </dl>
      <p className="season-totals-note">Full-time matches only · Goals include own goals.</p>
    </div>
  );
  const rounds = [...new Set(data.fixtures.map((f) => f.matchday))].sort(
    (a, b) => a - b,
  );
  const filteredFixtures = data.fixtures.filter(
    (f) =>
      (round === "all" || String(f.matchday) === round) &&
      (status === "all" || f.status === status) &&
      `${f.home_team} ${f.away_team} ${f.home_team_tag} ${f.away_team_tag}`
        .toLowerCase()
        .includes(search.toLowerCase()),
  );
  const filteredStats = data.stats.filter((p) =>
    `${p.name} ${p.team_name} ${p.jersey_number}`
      .toLowerCase()
      .includes(search.toLowerCase()),
  );
  const renderFixtures = (fixtures: Fixture[], limit?: number) =>
    !fixtures.length ? (
      <Empty
        title="No fixtures to show"
        detail={
          data.fixtures.length
            ? "Try another matchday, status, or team name."
            : "The season schedule will appear once an admin generates the fixtures."
        }
      />
    ) : (
      <div className="fixture-list">
        {fixtures.slice(0, limit).map((f) => (
          <FixtureRow
            key={f.id}
            fixture={f}
            onClick={() => setSelectedFixture(f)}
          />
        ))}
      </div>
    );
  const openPlayer = (p: PlayerStats) =>
    setExplorer({
      player: {
        id: p.player_id,
        name: p.name,
        jersey_number: p.jersey_number,
        team: p.team_name,
      },
    });
  if (adminArea && !token) return <AdminLogin onVerified={key => {
    setAdminCredential(key);
    setToken(key);
    window.history.replaceState(null, "", "/admin");
    setPage("admin");
    setAction(null);
  }} />;
  return (
    <div className={`app-shell${page==="membership"?" roaster-background":""}`}>
      <a className="skip-link" href="#main-content">
        Skip to content
      </a>
      <header className="site-header">
        <div className="site-header-inner">
          <a
            href="/"
            className="brand"
            
          >
            <img src="/crest.svg" alt="" />
            <span className="brand-name">
              GreatMinds<b>Sport</b>
            </span>
          </a>
          <ScrollNavigation>
            {[{id:"membership" as Page,label:"Roaster",active:page==="membership"},{id:"overview" as Page,label:"League",active:page!=="membership"&&page!=="admin"}].map(({id,label,active})=><a key={id} href={pageURL(id,adminArea)} className={active?"nav-item active":"nav-item"} aria-current={active?"page":undefined} onClick={event=>{event.preventDefault();navigate(id);}}>{label}</a>)}
            {adminArea && <a href="/admin" className={page === "admin" ? "nav-item active" : "nav-item"} aria-current={page === "admin" ? "page" : undefined} onClick={event => { event.preventDefault(); navigate("admin"); }}>Dashboard</a>}
          </ScrollNavigation>
          <div className="header-actions">
            
            <Tooltip title={mode === "dark" ? "Switch to light mode" : "Switch to dark mode"}>
              <IconButton className="theme-toggle" onClick={toggle} aria-label={mode === "dark" ? "Switch to light mode" : "Switch to dark mode"} aria-pressed={mode === "dark"}>
                {mode === "dark" ? <LightModeOutlined /> : <DarkModeOutlined />}
              </IconButton>
            </Tooltip>
            <IconButton
              size="small"
              className="refresh-button"
              aria-label="Refresh league data"
              onClick={() => {
                void refresh();
                void loadRoster();
              }}
              disabled={refreshing}
            >
              <RefreshRounded className={refreshing ? "spinning" : ""} />
            </IconButton>

          </div>
        </div>
      </header>
      <main id="main-content">
        {page!=="membership"&&page!=="admin"&&<section className="league-navigation" aria-label="RealEdehsonsLeague">
          <h1>RealEdehsonsLeague</h1>
          <ScrollNavigation id="league-navigation" label="League navigation">{pages.filter(p=>p.id!=="membership").map(({id,label})=><a key={id} href={pageURL(id,adminArea)} className={page===id?"nav-item active":"nav-item"} aria-current={page===id?"page":undefined} onClick={event=>{event.preventDefault();navigate(id);}}>{label}</a>)}</ScrollNavigation>
        </section>}

        {page !== "overview" && (
          <div className="page-title">
            <h1>{pageText[page].title}</h1>
            <p>{pageText[page].note}</p>
          </div>
        )}
        {errors.length > 0 && (
          <Alert
            severity="error"
            className="connection-alert"
            action={
              <Button color="inherit" onClick={() => void refresh()}>
                Retry
              </Button>
            }
          >
            <strong>Some league data could not be loaded.</strong>
            {Array.from(
              new Set(errors.map((e) => e.slice(e.indexOf(": ") + 2))),
            ).map((e) => (
              <div key={e}>{e}</div>
            ))}
            {updated && (
              <div>Showing previously loaded data where available.</div>
            )}
          </Alert>
        )}
        {loading ? (
          <div className="loading page-loading" role="status">
            <CircularProgress size={28} />
            <span>Loading the match centre…</span>
          </div>
        ) : (
          <div className="page" key={page}>
            {page === "membership" && <MembershipPage token={token} />}
            {page === "overview" && (
              <>
                <MatchFeature
                  fixture={featured}
                  isNext={!!next}
                  onOpen={() => featured && setSelectedFixture(featured)}
                  onAll={() => navigate("fixtures")}
                />
                {pendingResults.length > 0 && <Alert severity="info" action={<Button color="inherit" onClick={() => navigate("fixtures")}>View fixtures</Button>}>
                  {pendingResults.length} {pendingResults.length === 1 ? "match is" : "matches are"} awaiting a result. Scheduled kickoff has passed; scores are not yet published.
                </Alert>}
                <LiveBroadcastSection fixtures={data.fixtures} unavailable={unavailable("Fixtures")} onWatch={fixture => { setWatchFixtureID(fixture.id); setSelectedFixture(fixture); }} />
                <div className="overview-grid">
                  <section className="panel">
                    <SectionHeading
                      title="Table"
                      action={
                        <MoreButton onClick={() => navigate("standings")}>
                          Full table
                        </MoreButton>
                      }
                    />
                    <LeagueTable
                      rows={data.standings}
                      compact
                      onTeamClick={openTeam}
                    />
                    {seasonTotals}
                    <div className="panel-foot">
                      3 points for a win, 1 for a draw.
                    </div>
                  </section>

                </div>
                {upcoming.length > 1 && (
                  <section className="block">
                    <SectionHeading
                      title="Coming up"
                      action={
                        <MoreButton onClick={() => navigate("fixtures")}>
                          All fixtures
                        </MoreButton>
                      }
                    />
                    {renderFixtures(upcoming.slice(1), 3)}
                  </section>
                )}
                {latestResults.length > 0 && (
                  <section className="block">
                    <SectionHeading
                      title="Latest results"
                      action={
                        <MoreButton
                          onClick={() => {
                            setStatus("finalized");
                            navigate("fixtures");
                          }}
                        >
                          All results
                        </MoreButton>
                      }
                    />
                    {renderFixtures(latestResults, 3)}
                  </section>
                )}
              </>
            )}
            {page === "standings" && (
              <section className="panel">
                <LeagueTable rows={data.standings} onTeamClick={openTeam} />
                {seasonTotals}
                <LeagueLeaders standings={unavailable("Standings")?null:data.standings} stats={unavailable("Player statistics")?null:data.stats} onTeamClick={openTeam} />
                <div className="table-notes">
                  <p>
                    P played · W won · D drawn · L lost · GF goals for · GA goals
                    against · GD goal difference · Pts points
                  </p>
                  <p>
                    Only finalized matches count. Positions are numbered
                    consecutively.
                  </p>
                </div>
              </section>
            )}
            {page === "fixtures" && (
              <>
                <div className="filter-bar">
                  <TextField
                    label="Search teams"
                    value={search}
                    onChange={(e) => setSearch(e.target.value)}
                    slotProps={{
                      input: {
                        startAdornment: (
                          <SearchRounded className="search-icon" />
                        ),
                      },
                    }}
                  />
                  <Select
                    size="small"
                    value={round}
                    onChange={(e) => setRound(e.target.value)}
                    inputProps={{ "aria-label": "Filter by matchday" }}
                  >
                    <MenuItem value="all">All matchdays</MenuItem>
                    {rounds.map((r) => (
                      <MenuItem key={r} value={String(r)}>
                        Matchday {r}
                      </MenuItem>
                    ))}
                  </Select>
                  <Select
                    size="small"
                    value={status}
                    onChange={(e) => setStatus(e.target.value)}
                    inputProps={{ "aria-label": "Filter by match status" }}
                  >
                    <MenuItem value="all">All matches</MenuItem>
                    <MenuItem value="scheduled">Scheduled</MenuItem>
                    <MenuItem value="live">Live</MenuItem>
                    <MenuItem value="half_time">Half-time</MenuItem>
                    <MenuItem value="finalized">Final results</MenuItem>
                    <MenuItem value="postponed">Postponed</MenuItem>
                    <MenuItem value="abandoned">Abandoned</MenuItem>
                  </Select>
                </div>
                {filteredFixtures.length
                  ? rounds
                      .filter((r) =>
                        filteredFixtures.some((f) => f.matchday === r),
                      )
                      .map((r) => (
                        <section className="block" key={r}>
                          <SectionHeading
                            title={`Matchday ${r}`}
                            action={
                              <span className="muted small">
                                {
                                  filteredFixtures.filter(
                                    (f) => f.matchday === r,
                                  ).length
                                }{" "}
                                matches
                              </span>
                            }
                          />
                          {renderFixtures(
                            filteredFixtures.filter((f) => f.matchday === r),
                          )}
                        </section>
                      ))
                  : renderFixtures([])}
              </>
            )}
            {page === "players" && (
              <section className="panel">
                <SectionHeading
                  title="Player statistics"
                  action={
                    <span className="subtle-tag">Finalized matches only</span>
                  }
                />
                <div className="panel-search">
                  <TextField
                    label="Search player or team"
                    value={search}
                    onChange={(e) => setSearch(e.target.value)}
                  />
                </div>
                {filteredStats.length ? (
                  <ScrollTable>
                    <Table aria-label="Player statistics">
                      <TableHead>
                        <TableRow>
                          <TableCell>Player</TableCell>
                          <TableCell>Team</TableCell>
                          <TableCell align="center">
                            <span className="card-chip yellow" aria-hidden="true" />
                            Yellow
                          </TableCell>
                          <TableCell align="center">
                            <span className="card-chip red" aria-hidden="true" />
                            Red
                          </TableCell>
                        </TableRow>
                      </TableHead>
                      <TableBody>
                        {filteredStats.map((p) => (
                          <TableRow key={p.player_id}>
                            <TableCell>
                              <button
                                className="text-link"
                                onClick={() => openPlayer(p)}
                              >
                                {p.name}
                              </button>
                              <small className="cell-sub">
                                #{p.jersey_number}
                              </small>
                            </TableCell>
                            <TableCell>{p.team_name}</TableCell>
                            <TableCell align="center" className="num">
                              {p.yellow_cards}
                            </TableCell>
                            <TableCell align="center" className="num">
                              {p.red_cards}
                            </TableCell>
                          </TableRow>
                        ))}
                      </TableBody>
                    </Table>
                  </ScrollTable>
                ) : (
                  <Empty
                    title="No players found"
                    detail={
                      search
                        ? "Try another player or team name."
                        : "Registered players and their match statistics will appear here."
                    }
                  />
                )}
                <div className="panel-foot">
                  Yellow and red cards from finalized matches.
                </div>
              </section>
            )}
            {page === "teams" && (
              <>
                {token && <div className="team-management-toolbar">
                  <Button variant="contained" startIcon={<AddRounded />} onClick={() => adminAction("team")}>Add team</Button>
                </div>}
                <div className="team-picker" role="group" aria-label="Choose a team">
                  {data.teams.map((t) => (
                    <button
                      key={t.tag}
                      className={`team-tile ${selectedTeam === t.tag ? "selected" : ""}`}
                      aria-pressed={selectedTeam === t.tag}
                      onClick={() => setSelectedTeam(t.tag)}
                    >
                      <Crest tag={t.tag} name={t.name} size={40} />
                      <span className="team-tile-text">
                        <strong>{t.name}</strong>
                        <small>{t.tag}</small>
                      </span>
                    </button>
                  ))}
                </div>
                {!data.teams.length ? (
                  <Empty
                    title="No teams yet"
                    detail="Registered teams and their rosters will appear here."
                  />
                ) : (
                  <section className="panel">
                    <SectionHeading
                      title={`${data.teams.find((t) => t.tag === selectedTeam)?.name ?? "Team"} squad`}
                      action={
                        token ? (
                          <div style={{display:"flex", flexWrap:"wrap", gap:8}}>
                            <Button startIcon={<EditOutlined />} onClick={() => setRenamingTeam(true)}>Edit team</Button>
                            <Button startIcon={<AddRounded />} onClick={() => adminAction("player")}>Add player</Button>
                          </div>
                        ) : undefined
                      }
                    />
                    <div className="team-summary-panel">
                      <TeamSummary teamTag={selectedTeam} standings={unavailable("Standings") ? null : data.standings} stats={unavailable("Player statistics") ? null : data.stats} onPlayerClick={openPlayer} />
                    </div>
                    {rosterError && (
                      <Alert
                        severity="error"
                        action={
                          <Button onClick={() => void loadRoster()}>
                            Retry
                          </Button>
                        }
                      >
                        {rosterError}
                      </Alert>
                    )}
                    {rosterLoading ? (
                      <div className="loading">
                        <CircularProgress size={24} />
                      </div>
                    ) : roster.length ? (
                      <>
                        <div className="roster-captain-summary">
                          <MilitaryTechOutlined fontSize="small" />
                          {roster.find((p) => p.is_captain) ? (
                            <span>
                              Captain{" "}
                              <strong>
                                {roster.find((p) => p.is_captain)!.name}
                              </strong>
                            </span>
                          ) : (
                            <span>
                              Captain not assigned
                              {token
                                ? " · Use a player’s captain button to assign one."
                                : ""}
                            </span>
                          )}
                        </div>
                        <div
                          className={`roster-grid ${token ? "roster-admin" : ""}`}
                        >
                          {roster.map((p) => (
                            <div
                              className={`roster-player ${p.is_captain ? "roster-captain" : ""}`}
                              key={p.id}
                            >
                              <span className="jersey">{p.jersey_number}</span>
                              <div className="roster-player-info">
                                <div className="player-name-line">
                                  <button
                                    className="text-link"
                                    onClick={() =>
                                      setExplorer({
                                        player: {
                                          id: p.id,
                                          name: p.name,
                                          jersey_number: p.jersey_number,
                                          team: p.team,
                                        },
                                      })
                                    }
                                  >
                                    {p.name}
                                  </button>
                                  {p.is_captain && (
                                    <span className="captain-badge">
                                      <b>C</b> Captain
                                    </span>
                                  )}
                                </div>
                              </div>
                              {token && (
                                <div className="roster-actions">
                                  <Tooltip title={`Edit ${p.name}'s name or jersey number`}>
                                    <Button startIcon={<EditOutlined fontSize="small" />}
                                      size="small"
                                      aria-label={`Edit ${p.name}`}
                                      onClick={() =>
                                        setPlayerAction({
                                          player: p,
                                          teamTag: selectedTeam,
                                          action: "edit",
                                        })
                                      }
                                    >
                                      Edit player
                                    </Button>
                                  </Tooltip>
                                  <Tooltip
                                    title={
                                      p.is_captain
                                        ? `${p.name} is the current captain`
                                        : `Make ${p.name} captain`
                                    }
                                  >
                                    <span>
                                      <IconButton
                                        size="small"
                                        disabled={p.is_captain}
                                        aria-label={`Make ${p.name} captain`}
                                        onClick={() =>
                                          setPlayerAction({
                                            player: p,
                                            teamTag: selectedTeam,
                                            action: "captain",
                                          })
                                        }
                                      >
                                        <MilitaryTechOutlined fontSize="small" />
                                      </IconButton>
                                    </span>
                                  </Tooltip>
                                  <Tooltip
                                    title={
                                      p.is_captain
                                        ? "Assign another captain before removal"
                                        : `Remove ${p.name}`
                                    }
                                  >
                                    <span>
                                      <IconButton
                                        size="small"
                                        disabled={p.is_captain}
                                        aria-label={`Remove ${p.name}`}
                                        onClick={() =>
                                          setPlayerAction({
                                            player: p,
                                            teamTag: selectedTeam,
                                            action: "remove",
                                          })
                                        }
                                      >
                                        <DeleteOutlineRounded fontSize="small" />
                                      </IconButton>
                                    </span>
                                  </Tooltip>
                                </div>
                              )}
                            </div>
                          ))}
                        </div>
                      </>
                    ) : (
                      !rosterError && (
                        <Empty
                          title="No players yet"
                          detail="The league admin can add players to this team."
                        />
                      )
                    )}
                  </section>
                )}
              </>
            )}
            {page === "admin" && (
              <>
                <section className="admin-intro panel">
                  <div><h2>Administration</h2><p>You are signed in for this session.</p></div>
                  <Button startIcon={<LogoutRounded />} onClick={() => {
                    setAdminCredential(""); setToken(""); setAction(null); setSelectedFixture(null); setRenamingTeam(false); setPlayerAction(null); navigate("admin");
                  }}>Log out</Button>
                </section>
                <ViewerPINSettings token={token}/>
                <div className="admin-grid">
                  <AdminCard
                    title="Team management"
                    detail="Manage team, player details, jersey numbers, and team captains."
                    count={data.teams.length}
                    unit="teams"
                    button="Manage team"
                    onClick={() => navigate("teams")}
                  />
                  <AdminCard
                    title="Season schedule"
                    detail="Generate fixtures or add missing pairings after registering new teams."
                    count={data.fixtures.length}
                    unit="fixtures"
                    button={
                      data.fixtures.length
                        ? "Update schedule"
                        : "Generate fixtures"
                    }
                    onClick={() => adminAction("generate")}
                  />
                </div>
                <section className="panel membership-admin-entry">
                  <div><h2>Great Minds Sport</h2><p>Manage independent members, monthly dues, card fines, fee rates and record history.</p></div>
                  <Button variant="contained" onClick={() => navigate("membership")}>Manage membership</Button>
                </section>
                <section className="panel">
                  <SectionHeading title="Schedule completeness" />
                  <div className="schedule-completeness-body">
                  <div className="schedule-completeness-counts">
                    <span><strong>{data.fixtures.length} of {schedule.expected}</strong> fixtures</span>
                    <span><strong>{new Set(data.fixtures.map(f => f.matchday)).size} of {schedule.expectedDays}</strong> matchdays</span>
                  </div>
                  {data.teams.length !== 4 && <p className="muted">{data.teams.length} of 4 teams registered. Register all four to complete the league schedule.</p>}
                  {schedule.unexpectedDays.length > 0 || (data.fixtures.length >= schedule.expected && schedule.incomplete.length > 0) ? <Alert severity="warning">Existing schedule needs review. {schedule.unexpectedDays.length > 0 && `Unexpected matchdays: ${schedule.unexpectedDays.join(", ")}. `}The schedule updater preserves existing fixtures and cannot rearrange this layout.</Alert> : schedule.incomplete.length > 0 ? <Alert severity="info">Incomplete matchdays: {schedule.incomplete.join(", ")}. Update schedule adds missing pairings where they fit; conflicting existing fixtures require review.</Alert> : data.fixtures.length > 0 ? <p>Every matchday has {Math.floor(data.teams.length / 2)} {data.teams.length === 4 ? "games" : "game"}.</p> : null}
                  </div>
                </section>
                <section className="panel">
                  <SectionHeading
                    title="Matchday management"
                    action={
                      <MoreButton onClick={() => navigate("fixtures")}>
                        Browse fixtures
                      </MoreButton>
                    }
                  />
                  <p className="panel-description">
                    Select a match to set its kickoff and venue, enter goals and
                    cards, or correct a published result.
                  </p>
                  {renderFixtures(data.fixtures)}
                </section>
                <Alert severity="info">
                  Use Teams to edit rosters and assign captains. Players with
                  match records are protected from removal.
                </Alert>
              </>
            )}
          </div>
        )}
      </main>
      <footer className="site-footer">
        <section className="footer-partners" aria-labelledby="partners-heading">
          <h2 id="partners-heading">Happy Hour</h2>
          <ul>
            <li>G-Gozil Garden Kubwa</li>
          </ul>
        </section>
        <div className="site-footer-inner">
          <span className="footer-brand">
            <img src="/crest.svg" alt="" />
            GreatMindsSport · Home of Talents
          </span>
          <span>
            {updated
              ? `Updated ${updated.toLocaleTimeString("en-NG", { hour: "2-digit", minute: "2-digit", timeZone: "Africa/Lagos" })} WAT`
              : "All match times in WAT (UTC+1)"}
          </span>
        </div>
      </footer>
      {explorer && <TeamExplorer selection={explorer} standings={unavailable("Standings") ? null : data.standings} seasonStats={unavailable("Player statistics") ? null : data.stats} onClose={() => setExplorer(null)} />}
      {action && (
        <AdminForms
          key={action}
          action={action}
          onClose={() => setAction(null)}
          token={token}
          onToken={() => {
            navigate("admin");
          }}
          teams={data.teams}
          fixtures={data.fixtures}
          defaultTeam={page === "teams" ? selectedTeam : undefined}
          onSaved={saved}
        />
      )}
      {selectedFixture && (
        <FixtureDialog
          key={`${selectedFixture.id}-${!!token}`}
          fixture={selectedFixture}
          teams={data.teams}
          initialWatch={watchFixtureID === selectedFixture.id}
          standings={unavailable("Standings") ? null : data.standings}
          seasonStats={unavailable("Player statistics") ? null : data.stats}
          onPlayerClick={player => { setSelectedFixture(null); openPlayer(player); }}
          token={token}
          onClose={() => { setSelectedFixture(null); setWatchFixtureID(""); }}
          onSaved={saved}
        />
      )}
      {renamingTeam && token && data.teams.find(t => t.tag === selectedTeam) && (
        <TeamRenameDialog key={selectedTeam} team={data.teams.find(t => t.tag === selectedTeam)!} token={token} onClose={() => setRenamingTeam(false)} onSaved={saved} />
      )}
      {playerAction && token && (
        <PlayerManagementDialog
          key={`${playerAction.player.id}-${playerAction.action}`}
          player={playerAction.player}
          teamTag={playerAction.teamTag}
          teamName={
            data.teams.find((t) => t.tag === playerAction.teamTag)?.name ??
            playerAction.teamTag
          }
          captainName={roster.find((p) => p.is_captain)?.name}
          action={playerAction.action}
          token={token}
          onClose={() => setPlayerAction(null)}
          onSaved={saved}
        />
      )}
      <Snackbar
        open={!!toast}
        autoHideDuration={6000}
        onClose={() => setToast("")}
        message={toast}
      />
    </div>
  );
}
function Stat({
  label,
  value,
  foot,
}: {
  label: string;
  value: number | null;
  foot?: string;
}) {
  return (
    <div className="figure">
      <dt>{label}</dt>
      <dd>
        <strong>{value === null ? "—" : value.toLocaleString()}</strong>
        {foot && <small>{foot}</small>}
      </dd>
    </div>
  );
}
function MatchFeature({
  fixture: f,
  isNext,
  onOpen,
  onAll,
}: {
  fixture?: Fixture;
  isNext: boolean;
  onOpen: () => void;
  onAll: () => void;
}) {
  if (!f)
    return (
      <section className="feature feature-empty">
        <h2>RealEdehsonsLeague</h2>
        <p>
          Home of Talents. The schedule will appear here once an admin generates
          the fixtures.
        </p>
      </section>
    );
  const parts = kickoffParts(f.kickoff_at);
  const showScore = hasPublishedScore(f);
  const waiting = awaitingResult(f);
  return (
    <>
      <span className="visually-hidden">RealEdehsonsLeague match centre</span>
      <section
        className="feature"
        aria-label={["live", "half_time"].includes(f.status) ? matchStatusLabel(f.status) : waiting ? "Awaiting result" : isNext ? (f.kickoff_at ? "Next match" : "Upcoming · date TBC") : "Latest result"}
      >
        <div className="feature-top">
          <span className="feature-label">
            {["live", "half_time"].includes(f.status) ? matchStatusLabel(f.status) : waiting ? "Awaiting result" : isNext ? (f.kickoff_at ? "Next match" : "Upcoming · date TBC") : "Latest result"}
          </span>
          <span>Matchday {f.matchday}</span>
        </div>
        <div className="feature-match">
          <div className="feature-team">
            <Crest tag={f.home_team_tag} name={f.home_team} size={88} />
            <strong>{f.home_team}</strong>
          </div>
          <div className="feature-centre">
            {showScore ? (
              <span className="feature-score">
                {f.home_score}
                <i>:</i>
                {f.away_score}
              </span>
            ) : (
              <span className="feature-vs">vs</span>
            )}
            <span className="feature-when">
              {parts ? (
                <>
                  {parts.date}
                  <br />
                  {parts.time} WAT
                </>
              ) : (
                "Kickoff to be confirmed"
              )}
            </span>
          </div>
          <div className="feature-team">
            <Crest tag={f.away_team_tag} name={f.away_team} size={88} />
            <strong>{f.away_team}</strong>
          </div>
        </div>
        <div className="feature-foot">
          <span className="feature-venue">
            <PlaceOutlined />
            {f.venue || "Venue to be confirmed"}
          </span>
          <span className="feature-actions">
            <Button variant="contained" color="secondary" onClick={onOpen}>
              Match details
            </Button>
            <Button
              className="feature-ghost"
              onClick={onAll}
              endIcon={<ArrowForwardRounded />}
            >
              All fixtures
            </Button>
          </span>
        </div>
      </section>
    </>
  );
}
function FixtureRow({
  fixture: f,
  onClick,
}: {
  fixture: Fixture;
  onClick: () => void;
}) {
  const parts = kickoffParts(f.kickoff_at);
  const done = f.status === "finalized";
  const showScore = hasPublishedScore(f);
  const waiting = awaitingResult(f);
  return (
    <button className="fixture-row" onClick={onClick}>
      <span className="fx-when">
        <span className="fx-day">{parts ? parts.date : "Date TBC"}</span>
        <span className="fx-md">Matchday {f.matchday}</span>
      </span>
      <span className="fx-team fx-home">
        <strong>{f.home_team}</strong>
        <Crest tag={f.home_team_tag} name={f.home_team} size={30} />
      </span>
      <span className="fx-mid">
        {showScore ? (
          <span className="fx-score">
            {f.home_score} – {f.away_score}
          </span>
        ) : (
          <span className="fx-time">{parts ? parts.time : "vs"}</span>
        )}
        {(f.status !== "scheduled" || waiting) && (
          <span className={`fx-status ${f.status}`}>
            {done ? (
              <abbr title="Full time">FT</abbr>
            ) : (
              waiting ? "Awaiting result" : matchStatusLabel(f.status)
            )}
          </span>
        )}
      </span>
      <span className="fx-team fx-away">
        <Crest tag={f.away_team_tag} name={f.away_team} size={30} />
        <strong>{f.away_team}</strong>
      </span>
      <span className="fx-venue">
        <PlaceOutlined />
        {f.venue || "Venue TBC"}
      </span>
    </button>
  );
}
function AdminCard({
  title,
  detail,
  count,
  unit,
  button,
  onClick,
  disabled,
}: {
  title: string;
  detail: string;
  count: number;
  unit: string;
  button: string;
  onClick: () => void;
  disabled?: boolean;
}) {
  return (
    <section className="panel admin-card">
      <div className="admin-card-count">
        <strong>{count}</strong> {unit}
      </div>
      <h2>{title}</h2>
      <p>{detail}</p>
      <Button variant="outlined" onClick={onClick} disabled={disabled}>
        {button}
      </Button>
    </section>
  );
}
