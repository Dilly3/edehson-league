import { useRef, useState, type FormEvent } from "react";
import { Alert, Button, Dialog, DialogActions, DialogContent, DialogTitle, InputAdornment, MenuItem, TextField } from "@mui/material";
import { api, message } from "../api";
import { money, nairaToKobo, watDate, type Member, type RegisterData } from "../membership";

/** Admin: record a donation from a player (front page or player profile). */
export default function RecordDonation({ data, token, initialMember, onClose, onSaved }: {
  data: RegisterData; token: string; initialMember?: Member; onClose: () => void; onSaved: () => void;
}) {
  const [memberID, setMemberID] = useState(initialMember?.id ?? "");
  const [amount, setAmount] = useState("");
  const [date, setDate] = useState(watDate());
  const [purpose, setPurpose] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const retry = useRef<{ body: string; id: string } | null>(null);
  const member = data.members.find(m => m.id === memberID);
  let kobo: number | null = null;
  try { kobo = amount.trim() ? nairaToKobo(amount.replace(/,/g, "")) : null; } catch { kobo = null; }

  const submit = async (e: FormEvent) => {
    e.preventDefault();
    if (busy) return;
    setError("");
    if (!member) { setError("Select the player making the donation."); return; }
    if (kobo === null) { setError("Enter a donation amount greater than zero, with at most two decimal places."); return; }
    if (!date || date < member.joined_on || date > watDate()) { setError(`Choose a date between ${member.joined_on} (joined) and today.`); return; }
    if (!purpose.trim()) { setError("Enter what the donation is for."); return; }
    const body = { type: "add_donation", member_id: member.id, amount: kobo, paid_on: date, note: purpose.trim() };
    const fingerprint = JSON.stringify(body);
    if (retry.current?.body !== fingerprint) retry.current = { body: fingerprint, id: crypto.randomUUID() };
    setBusy(true);
    try { await api("/membership/commands", { method: "POST", body: JSON.stringify({ ...body, request_id: retry.current.id }) }, token); onSaved(); }
    catch (e) { setError(message(e)); }
    finally { setBusy(false); }
  };

  return <Dialog open onClose={busy ? undefined : onClose} maxWidth="sm" fullWidth><form onSubmit={submit}>
    <DialogTitle>Record donation</DialogTitle>
    <DialogContent><div className="dialog-form">
      {error && <Alert severity="error">{error}</Alert>}
      <TextField select label="Player" value={memberID} onChange={e => { setMemberID(e.target.value); setError(""); }} required disabled={busy || !!initialMember}>
        <MenuItem value="" disabled>Select a player</MenuItem>
        {data.members.map(m => <MenuItem key={m.id} value={m.id}>{m.name}{m.active ? "" : " · Inactive"}</MenuItem>)}
      </TextField>
      <TextField label="Amount" value={amount} onChange={e => setAmount(e.target.value)} required disabled={busy} inputMode="decimal" autoComplete="off"
        helperText={kobo !== null ? `Records ${money(kobo)}` : "In naira, e.g. 5000"}
        slotProps={{ input: { startAdornment: <InputAdornment position="start">₦</InputAdornment> } }}/>
      <TextField label="Date received" type="date" value={date} onChange={e => setDate(e.target.value)} required disabled={busy}
        slotProps={{ inputLabel: { shrink: true }, htmlInput: { min: member?.joined_on, max: watDate() } }}/>
      <TextField label="Donation purpose" value={purpose} onChange={e => setPurpose(e.target.value)} required disabled={busy} multiline minRows={2}
        slotProps={{ htmlInput: { maxLength: 300 } }} helperText="e.g. New kits, pitch hire, end-of-season party. Visible to members."/>
    </div></DialogContent>
    <DialogActions>
      <Button onClick={onClose} disabled={busy}>Cancel</Button>
      <Button type="submit" variant="contained" disabled={busy || !member || kobo === null || !date || !purpose.trim()}>{busy ? "Recording…" : "Record donation"}</Button>
    </DialogActions>
  </form></Dialog>;
}
