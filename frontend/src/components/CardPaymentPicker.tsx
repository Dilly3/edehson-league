import { useState } from "react";
import { Button, Dialog, DialogActions, DialogContent, DialogTitle, MenuItem, TextField } from "@mui/material";
import { cardSummary, money, type Member, type RegisterData } from "../membership";

/** Front-page entry to the card-payment flow: pick a player who owes card fines,
 * then continue to the existing "Settle card fines" form for that player. */
export default function CardPaymentPicker({ data, onClose, onContinue }: {
  data: RegisterData; onClose: () => void; onContinue: (member: Member) => void;
}) {
  const owing = data.members
    .map(m => ({ m, y: cardSummary(data, m.id, "yellow"), r: cardSummary(data, m.id, "red") }))
    .filter(x => x.y.outstanding + x.r.outstanding > 0)
    .sort((a, b) => a.m.name.localeCompare(b.m.name));
  const [memberID, setMemberID] = useState("");
  const chosen = owing.find(x => x.m.id === memberID);
  return <Dialog open onClose={onClose} maxWidth="sm" fullWidth>
    <form onSubmit={e => { e.preventDefault(); if (chosen) onContinue(chosen.m); }}>
      <DialogTitle>Record card payment</DialogTitle>
      <DialogContent><div className="dialog-form">
        {owing.length === 0 ? <p className="membership-note">No player has unpaid yellow or red cards.</p> : <>
          <TextField select label="Player" value={memberID} onChange={e => setMemberID(e.target.value)} required autoFocus
            helperText="Only players with unpaid cards are listed.">
            {owing.map(({ m, y, r }) => <MenuItem key={m.id} value={m.id}>
              {m.name} — {[y.outstanding && `${y.outstanding} yellow`, r.outstanding && `${r.outstanding} red`].filter(Boolean).join(", ")} · {money(y.balance + r.balance)}
            </MenuItem>)}
          </TextField>
          {chosen && <p className="muted">Owes {money(chosen.y.balance)} in yellow-card fines and {money(chosen.r.balance)} in red-card fines. You’ll choose the card type and how many are being paid next.</p>}
        </>}
      </div></DialogContent>
      <DialogActions>
        <Button onClick={onClose}>Cancel</Button>
        <Button type="submit" variant="contained" disabled={!chosen}>Continue</Button>
      </DialogActions>
    </form>
  </Dialog>;
}
