import { useState, type FormEvent } from "react";
import { Alert, Button, Dialog, DialogActions, DialogContent, DialogTitle, TextField } from "@mui/material";
import type { Team } from "../types";
import { api, message } from "../api";

export default function TeamRenameDialog({ team, token, onClose, onSaved }: {
  team: Team;
  token: string;
  onClose: () => void;
  onSaved: () => void;
}) {
  const [name, setName] = useState(team.name);
  const [displayName, setDisplayName] = useState(team.display_name ?? "");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const submit = async (event: FormEvent) => {
    event.preventDefault();
    setError("");
    const trimmed = name.trim();
    if (Array.from(trimmed).length < 2 || Array.from(trimmed).length > 20) {
      setError("Team name must contain 2–20 characters."); return;
    }
    if (Array.from(displayName.trim()).length > 12) { setError("Display name must contain at most 12 characters."); return; }
    if (!token) { setError("Enter your admin key before renaming a team."); return; }
    setBusy(true);
    try {
      await api(`/teams/${encodeURIComponent(team.tag)}/name`, { method: "PATCH", body: JSON.stringify({ name: trimmed, display_name: displayName.trim() || null }) }, token);
      onSaved();
      onClose();
    } catch (err) { setError(message(err)); }
    finally { setBusy(false); }
  };
  return <Dialog open onClose={busy ? undefined : onClose} maxWidth="sm">
    <form onSubmit={submit}>
      <DialogTitle>Edit team</DialogTitle>
      <DialogContent><div className="dialog-form">
        {error && <Alert severity="error">{error}</Alert>}
        <TextField label="Team name" value={name} onChange={e => setName(e.target.value)} required autoFocus disabled={busy} slotProps={{htmlInput:{minLength:2,maxLength:20}}} />
        <TextField label="Display name (optional)" value={displayName} onChange={e => setDisplayName(e.target.value)} disabled={busy} helperText="Up to 12 characters, e.g. Man Utd. Leave empty to use the full name in match events." slotProps={{htmlInput:{maxLength:12}}} />
        <p className="muted">The new name appears throughout the league. The team’s players, fixtures, results and tag ({team.tag}) stay linked.</p>
      </div></DialogContent>
      <DialogActions><Button onClick={onClose} disabled={busy}>Cancel</Button><Button type="submit" variant="contained" disabled={busy || (name.trim() === team.name && displayName.trim() === (team.display_name ?? ""))}>{busy ? "Saving…" : "Save team"}</Button></DialogActions>
    </form>
  </Dialog>;
}
