import { useState, type FormEvent } from "react";
import {
  Alert,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  TextField,
} from "@mui/material";
import type { Player } from "../types";
import { assignCaptain, message, removePlayer, updatePlayer } from "../api";
export type PlayerAction = "edit" | "captain" | "remove";
export default function PlayerManagementDialog({
  player,
  teamTag,
  teamName,
  captainName,
  action,
  token,
  onClose,
  onSaved,
}: {
  player: Player;
  teamTag: string;
  teamName: string;
  captainName?: string;
  action: PlayerAction;
  token: string;
  onClose: () => void;
  onSaved: () => void;
}) {
  const [name, setName] = useState(player.name);
  const [jersey, setJersey] = useState(player.jersey_number);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const submit = async (event: FormEvent) => {
    event.preventDefault();
    setError("");
    if (!token) {
      setError("Set your admin key before changing the roster.");
      return;
    }
    if (action === "edit") {
      const length = Array.from(name.trim()).length;
      if (length < 2 || length > 20) {
        setError("Player name must contain 2–20 characters.");
        return;
      }
      if (!/^\d+$/.test(jersey) || Number(jersey) < 1 || Number(jersey) > 100) {
        setError("Choose a jersey number from 1 to 100.");
        return;
      }
    }
    setBusy(true);
    try {
      if (action === "edit")
        await updatePlayer(
          teamTag,
          player.id,
          { name: name.trim(), jersey_number: Number(jersey) },
          token,
        );
      if (action === "captain") await assignCaptain(teamTag, player.id, token);
      if (action === "remove") await removePlayer(teamTag, player.id, token);
      onSaved();
      onClose();
    } catch (err) {
      setError(message(err));
    } finally {
      setBusy(false);
    }
  };
  const blocked = action === "remove" && player.is_captain;
  return (
    <Dialog open onClose={busy ? undefined : onClose} maxWidth="sm">
      <form onSubmit={submit}>
        <DialogTitle>
          {action === "edit"
            ? "Edit player"
            : action === "captain"
              ? "Assign team captain"
              : "Remove player"}
        </DialogTitle>
        <DialogContent>
          <div className="dialog-form">
            <p>
              <strong>{player.name}</strong> · {teamName} · #
              {player.jersey_number}
            </p>
            {error && <Alert severity="error">{error}</Alert>}
            {action === "edit" && (
              <>
                <TextField
                  label="Player name"
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  required
                  autoFocus
                  disabled={busy}
                  slotProps={{ htmlInput: { minLength: 2, maxLength: 20 } }}
                />
                <TextField
                  label="Jersey number"
                  type="number"
                  value={jersey}
                  onChange={(e) => setJersey(e.target.value)}
                  required
                  disabled={busy}
                  slotProps={{ htmlInput: { min: 1, max: 100, step: 1 } }}
                />
                <p>The player stays on {teamName}.</p>
              </>
            )}
            {action === "captain" && (
              <>
                <p>
                  Make <strong>{player.name}</strong> the captain of{" "}
                  <strong>{teamName}</strong>?
                </p>
                <Alert severity="info">
                  {captainName
                    ? `${captainName} will no longer be captain. This change takes effect immediately.`
                    : "This team currently has no assigned captain."}
                </Alert>
              </>
            )}
            {action === "remove" && (
              <>
                <p>
                  Remove <strong>{player.name}</strong> from this roster?
                </p>
                <Alert severity={blocked ? "warning" : "info"}>
                  {blocked
                    ? "Assign another captain before removing this player."
                    : "Players with match records cannot be removed. The league keeps those records intact."}
                </Alert>
                {!blocked && (
                  <p>
                    The player will disappear from the active roster. Their
                    record is retained. Restoring removed players is not
                    available in this app yet.
                  </p>
                )}
              </>
            )}
          </div>
        </DialogContent>
        <DialogActions>
          <Button disabled={busy} onClick={onClose}>
            Cancel
          </Button>
          <Button
            variant="contained"
            color={action === "remove" ? "error" : "primary"}
            type="submit"
            disabled={busy || blocked}
          >
            {busy
              ? "Saving…"
              : action === "edit"
                ? "Save player"
                : action === "captain"
                  ? "Assign captain"
                  : "Remove player"}
          </Button>
        </DialogActions>
      </form>
    </Dialog>
  );
}
