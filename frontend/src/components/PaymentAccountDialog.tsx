import { useEffect, useState, type FormEvent } from "react";
import { Alert, Button, CircularProgress, Dialog, DialogActions, DialogContent, DialogTitle, TextField } from "@mui/material";
import { message } from "../api";
import { getPaymentAccount, savePaymentAccount } from "../payments";

/** Roaster admin: the bank account players transfer to. */
export default function PaymentAccountDialog({ token, onClose, onSaved }: { token: string; onClose: () => void; onSaved: () => void }) {
  const [bank, setBank] = useState(""), [name, setName] = useState(""), [number, setNumber] = useState("");
  const [loading, setLoading] = useState(true), [busy, setBusy] = useState(false), [error, setError] = useState("");
  useEffect(() => {
    void getPaymentAccount().then(a => { if (a) { setBank(a.bank_name); setName(a.account_name); setNumber(a.account_number); } })
      .catch(e => setError(message(e))).finally(() => setLoading(false));
  }, []);
  const submit = async (e: FormEvent) => {
    e.preventDefault();
    if (busy) return;
    const digits = number.replace(/\s/g, "");
    if (!/^\d{10}$/.test(digits)) { setError("Account numbers have exactly 10 digits."); return; }
    setBusy(true); setError("");
    try { await savePaymentAccount(token, { bank_name: bank.trim(), account_name: name.trim(), account_number: digits }); onSaved(); }
    catch (e) { setError(message(e)); }
    finally { setBusy(false); }
  };
  return <Dialog open onClose={busy ? undefined : onClose} maxWidth="xs" fullWidth><form onSubmit={submit}>
    <DialogTitle>Payment account</DialogTitle>
    <DialogContent><div className="dialog-form">
      <p className="muted">Players see these details when they tap Make payment.</p>
      {error && <Alert severity="error">{error}</Alert>}
      {loading ? <div className="loading" role="status"><CircularProgress size={18}/><span>Loading…</span></div> : <>
        <TextField label="Bank" value={bank} onChange={e => setBank(e.target.value)} required disabled={busy} slotProps={{ htmlInput: { maxLength: 60 } }}/>
        <TextField label="Account name" value={name} onChange={e => setName(e.target.value)} required disabled={busy} slotProps={{ htmlInput: { maxLength: 80 } }}/>
        <TextField label="Account number" value={number} onChange={e => setNumber(e.target.value.replace(/[^\d\s]/g, ""))} required disabled={busy} inputMode="numeric" helperText="10 digits"
          slotProps={{ htmlInput: { maxLength: 13 } }}/>
      </>}
    </div></DialogContent>
    <DialogActions><Button onClick={onClose} disabled={busy}>Cancel</Button><Button type="submit" variant="contained" disabled={busy || loading}>{busy ? "Saving…" : "Save account"}</Button></DialogActions>
  </form></Dialog>;
}
