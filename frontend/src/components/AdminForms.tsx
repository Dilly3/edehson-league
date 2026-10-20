import { useState, type FormEvent } from "react";
import {
  Alert,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  MenuItem,
  TextField,
} from "@mui/material";
import type { Team } from "../types";
import { api, message } from "../api";
export type AdminAction = "team" | "player" | "generate" | "key" | null;
export default function AdminForms({
  action,
  onClose,
  token,
  onToken,
  teams,
  defaultTeam,
  fixtures,
  onSaved,
}: {
  action: AdminAction;
  onClose: () => void;
  token: string;
  onToken: (key: string) => void;
  teams: Team[];
  defaultTeam?: string;
  fixtures: import("../types").Fixture[];
  onSaved: () => void;
}) {
  const [name, setName] = useState(""),
    [tag, setTag] = useState(""),
    [team, setTeam] = useState(defaultTeam || teams[0]?.tag || ""),
    [jersey, setJersey] = useState(""),
    [key, setKey] = useState(token),
    [busy, setBusy] = useState(false),
    [error, setError] = useState("");
  const existingPairs = new Set(fixtures.map(f => `${f.home_team_tag}:${f.away_team_tag}`));
  const missingCount = teams.reduce((count, home) => count + teams.filter(away => away.tag !== home.tag && !existingPairs.has(`${home.tag}:${away.tag}`)).length, 0);
  const submit = async (e: FormEvent) => {
    e.preventDefault();
    setError("");
    if (action === "key") {
      if (key.trim().length < 32) {
        setError(
          "Enter an admin key with at least 32 characters.",
        );
        return;
      }
      onToken(key.trim());
      onClose();
      return;
    }
    if (!token) {
      setError("Set your admin key before saving.");
      return;
    }
    setBusy(true);
    try {
      if (action === "team")
        await api(
          "/teams",
          {
            method: "POST",
            body: JSON.stringify({
              name: name.trim(),
              tag: tag.trim().toUpperCase(),
            }),
          },
          token,
        );
      if (action === "player")
        await api(
          `/teams/${encodeURIComponent(team)}/players`,
          {
            method: "POST",
            body: JSON.stringify({
              name: name.trim(),
              jersey_number: Number(jersey),
            }),
          },
          token,
        );
      if (action === "generate")
        await api("/fixtures/generate", { method: "POST" }, token);
      onSaved();
      onClose();
    } catch (e) {
      setError(message(e));
    } finally {
      setBusy(false);
    }
  };
  const titles = {
    key: "Enter admin key",
    team: "Register a team",
    player: "Add a player",
    generate: fixtures.length ? "Update season schedule" : "Generate season schedule",
  };
  return (
    <Dialog open={!!action} onClose={busy ? undefined : onClose} maxWidth="sm">
      <form onSubmit={submit}>
        <DialogTitle>{action ? titles[action] : ""}</DialogTitle>
        <DialogContent>
          <div className="dialog-form">
            {error && <Alert severity="error">{error}</Alert>}
            {action === "key" && (
              <>
                <TextField
                  label="Admin key"
                  helperText="Enter your admin key (at least 32 characters) for administrative access."
                  type="password"
                  value={key}
                  onChange={(e) => setKey(e.target.value)}
                  required
                  autoComplete="off"
                  autoFocus
                />
              </>
            )}
            {action === "team" && (
              <>
                <TextField
                  label="Team name"
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  required
                  autoFocus
                  slotProps={{ htmlInput: { minLength: 2, maxLength: 20 } }}
                />
                <TextField
                  label="Team tag"
                  value={tag}
                  onChange={(e) => setTag(e.target.value.toUpperCase())}
                  placeholder="EDL001"
                  helperText="EDL001 through EDL999. Tags must be unique."
                  required
                  slotProps={{
                    htmlInput: { pattern: "EDL(?!000)[0-9]{3}", maxLength: 6 },
                  }}
                />
              </>
            )}
            {action === "player" && (
              <>
                <TextField
                  select
                  label="Team"
                  value={team}
                  onChange={(e) => setTeam(e.target.value)}
                  required
                >
                  {teams.map((t) => (
                    <MenuItem key={t.tag} value={t.tag}>
                      {t.name} · {t.tag}
                    </MenuItem>
                  ))}
                </TextField>
                <TextField
                  label="Player name"
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  required
                  slotProps={{ htmlInput: { minLength: 2, maxLength: 20 } }}
                />
                <TextField
                  label="Jersey number"
                  type="number"
                  value={jersey}
                  onChange={(e) => setJersey(e.target.value)}
                  required
                  slotProps={{ htmlInput: { min: 1, max: 100, step: 1 } }}
                />
                <small className="muted">
                  Your current API accepts jersey numbers 1–100.
                </small>
              </>
            )}
            {action === "generate" && (
              <>
                <p>
                  Add missing home-and-away fixtures for the{" "}
                  <strong>{teams.length} registered teams</strong>.
                </p>
                <div className="generate-total">
                  {missingCount} <span>new fixtures</span>
                </div>
                <p>
                  Existing fixtures and results stay unchanged. Missing pairings fill
                  available matchday slots. With four teams, the completed schedule
                  has six matchdays with two games each. New kickoff times and venues
                  remain unconfirmed.
                </p>
                <ul>
                  {teams.map((t) => (
                    <li key={t.tag}>
                      {t.name} <span className="muted">· {t.tag}</span>
                    </li>
                  ))}
                </ul>
              </>
            )}
          </div>
        </DialogContent>
        <DialogActions>
          <Button onClick={onClose} disabled={busy}>
            Cancel
          </Button>
          <Button
            type="submit"
            variant="contained"
            disabled={
              busy ||
              (action === "player" && !teams.length) ||
              (action === "generate" && (teams.length < 2 || teams.length > 4 || missingCount === 0))
            }
          >
            {busy
              ? "Saving…"
              : action === "key"
                ? "Set admin key"
                : action === "generate"
                  ? (fixtures.length ? "Update schedule" : "Generate fixtures")
                  : "Save"}
          </Button>
        </DialogActions>
      </form>
    </Dialog>
  );
}
