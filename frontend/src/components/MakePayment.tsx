import { useEffect, useMemo, useState, type FormEvent } from "react";
import { Alert, Button, Checkbox, CircularProgress, Dialog, DialogActions, DialogContent, DialogTitle, FormControlLabel, IconButton, InputAdornment, MenuItem, TextField, Tooltip } from "@mui/material";
import ContentCopyRounded from "@mui/icons-material/ContentCopyRounded";
import RemoveRounded from "@mui/icons-material/RemoveRounded";
import AddRounded from "@mui/icons-material/AddRounded";
import { message } from "../api";
import { cardSummary, duesMonthLabel, money, watDate, type Member, type RegisterData } from "../membership";
import { extraCardPrice, getPaymentAccount, getPendingReceipts, submitReceipt, type PaymentAccount, type PendingReceipt } from "../payments";

const MAX_BYTES = 5 * 1024 * 1024;
const shortMonth = (m: string) => duesMonthLabel(m).split("-")[1] ?? m;

/** Player flow: see the club account, transfer, then upload the receipt for an admin to confirm. */
export default function MakePayment({ data, initialMember, onClose }: { data: RegisterData; initialMember?: Member; onClose: () => void }) {
  const [account, setAccount] = useState<PaymentAccount | null | undefined>(undefined);
  const [accountError, setAccountError] = useState("");
  const [memberID, setMemberID] = useState(initialMember?.id ?? "");
  const [pending, setPending] = useState<PendingReceipt[]>([]);
  const [payDues, setPayDues] = useState(false);
  const [months, setMonths] = useState<string[]>([]);
  const [payCards, setPayCards] = useState(false);
  const [yellow, setYellow] = useState(0);
  const [red, setRed] = useState(0);
  const [amountSent, setAmountSent] = useState("");
  const [sentTouched, setSentTouched] = useState(false);
  const [paidOn, setPaidOn] = useState(watDate());
  const [note, setNote] = useState("");
  const [file, setFile] = useState<File | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [done, setDone] = useState(false);
  const [copied, setCopied] = useState(false);

  useEffect(() => { void getPaymentAccount().then(setAccount).catch(e => { setAccount(null); setAccountError(message(e)); }); }, []);
  useEffect(() => {
    setMonths([]); setYellow(0); setRed(0); setPending([]);
    if (!memberID) return;
    let live = true;
    void getPendingReceipts(memberID).then(p => { if (live) setPending(p); }).catch(() => {});
    return () => { live = false; };
  }, [memberID]);

  const pendingMonths = useMemo(() => new Set(pending.flatMap(p => p.months)), [pending]);
  const pendingYellow = pending.reduce((n, p) => n + p.yellow_count, 0);
  const pendingRed = pending.reduce((n, p) => n + p.red_count, 0);
  const unpaid = data.dues.filter(d => d.member_id === memberID && d.status === "unpaid").sort((a, b) => a.month.localeCompare(b.month));
  const now = watDate().slice(0, 7);
  const yellowLeft = memberID ? Math.max(0, cardSummary(data, memberID, "yellow").outstanding - pendingYellow) : 0;
  const redLeft = memberID ? Math.max(0, cardSummary(data, memberID, "red").outstanding - pendingRed) : 0;

  const chosenMonths = payDues ? months : [];
  const chosenYellow = payCards ? yellow : 0, chosenRed = payCards ? red : 0;
  const duesTotal = unpaid.filter(d => chosenMonths.includes(d.month)).reduce((n, d) => n + d.amount, 0);
  const yellowTotal = memberID ? extraCardPrice(data, memberID, "yellow", pendingYellow, chosenYellow) : 0;
  const redTotal = memberID ? extraCardPrice(data, memberID, "red", pendingRed, chosenRed) : 0;
  const total = duesTotal + (yellowTotal ?? 0) + (redTotal ?? 0);
  const hasItems = chosenMonths.length > 0 || chosenYellow > 0 || chosenRed > 0;

  useEffect(() => { if (!sentTouched) setAmountSent(total ? String(total / 100) : ""); }, [total, sentTouched]);

  const toggleMonth = (m: string) => setMonths(list => list.includes(m) ? list.filter(x => x !== m) : [...list, m].sort());
  const copy = async () => {
    if (!account) return;
    try { await navigator.clipboard.writeText(account.account_number); setCopied(true); setTimeout(() => setCopied(false), 1800); } catch { /* clipboard blocked */ }
  };

  const submit = async (e: FormEvent) => {
    e.preventDefault();
    if (busy) return;
    setError("");
    if (!memberID) { setError("Select your name."); return; }
    if (!hasItems) { setError("Choose what you paid for: monthly dues, cards, or both."); return; }
    if (!file) { setError("Attach your payment receipt."); return; }
    if (file.size > MAX_BYTES) { setError("The receipt must be 5 MB or smaller."); return; }
    if (!/^\d[\d,]*(\.\d{1,2})?$/.test(amountSent.trim())) { setError("Enter the amount you transferred."); return; }
    setBusy(true);
    try {
      await submitReceipt({ memberID, months: chosenMonths, yellow: chosenYellow, red: chosenRed, amountSent: amountSent.trim(), paidOn, note: note.trim(), file });
      setDone(true);
    } catch (e) { setError(message(e)); }
    finally { setBusy(false); }
  };

  const member = data.members.find(m => m.id === memberID);

  if (done) return <Dialog open onClose={onClose} maxWidth="sm" fullWidth>
    <DialogTitle>Receipt sent</DialogTitle>
    <DialogContent><div className="dialog-form">
      <Alert severity="success">Thanks{member ? `, ${member.name.split(" ")[0]}` : ""}. Your receipt for {money(total)} is waiting for an admin to check it. Your record updates once it’s confirmed.</Alert>
    </div></DialogContent>
    <DialogActions><Button variant="contained" onClick={onClose}>Done</Button></DialogActions>
  </Dialog>;

  return <Dialog open onClose={busy ? undefined : onClose} maxWidth="sm" fullWidth className="make-payment-dialog">
    <form onSubmit={submit}>
      <DialogTitle>Make payment</DialogTitle>
      <DialogContent><div className="dialog-form">
        <section className="pay-step">
          <h3><span>1</span> Transfer to the club account</h3>
          {account === undefined ? <div className="loading" role="status"><CircularProgress size={18}/><span>Loading account…</span></div>
            : account ? <div className="pay-account">
              <div><small>Bank</small><strong>{account.bank_name}</strong></div>
              <div><small>Account name</small><strong>{account.account_name}</strong></div>
              <div className="pay-account-number"><small>Account number</small>
                <strong>{account.account_number}</strong>
                <Tooltip title={copied ? "Copied" : "Copy account number"}><IconButton size="small" onClick={() => void copy()} aria-label="Copy account number"><ContentCopyRounded fontSize="small"/></IconButton></Tooltip>
              </div>
            </div>
            : <Alert severity="warning">{accountError || "The club hasn’t added its bank details yet. Please check with an admin before paying."}</Alert>}
        </section>

        <section className="pay-step">
          <h3><span>2</span> Tell us what you paid for</h3>
          {error && <Alert severity="error">{error}</Alert>}
          <TextField select label="Your name" value={memberID} onChange={e => setMemberID(e.target.value)} required disabled={busy || !!initialMember}>
            <MenuItem value="" disabled>Select your name</MenuItem>
            {data.members.filter(m => m.active || m.id === memberID).map(m => <MenuItem key={m.id} value={m.id}>{m.name}</MenuItem>)}
          </TextField>
          {pending.length > 0 && <Alert severity="info">Already awaiting review: {pending.map(p => [p.months.map(shortMonth).join(", "), p.yellow_count && `${p.yellow_count} yellow`, p.red_count && `${p.red_count} red`].filter(Boolean).join(" · ") + ` (${money(p.amount)})`).join("; ")}.</Alert>}

          {memberID && <>
            <FormControlLabel control={<Checkbox checked={payDues} onChange={e => setPayDues(e.target.checked)} disabled={busy || !unpaid.some(d => !pendingMonths.has(d.month))}/>} label="Monthly dues"/>
            {payDues && <div className="pay-months" role="group" aria-label="Months paid for">
              {unpaid.map(d => {
                const waiting = pendingMonths.has(d.month), on = months.includes(d.month);
                return <button key={d.id} type="button" className={`${on ? "on" : ""}${d.month > now ? " advance" : ""}`} aria-pressed={on} disabled={busy || waiting} onClick={() => toggleMonth(d.month)}
                  title={waiting ? "Awaiting review" : d.month > now ? "Advance payment" : undefined}>
                  <strong>{shortMonth(d.month)}</strong><small>{waiting ? "In review" : money(d.amount)}</small>
                </button>;
              })}
              {!unpaid.length && <p className="muted">No unpaid months for this year.</p>}
            </div>}

            <FormControlLabel control={<Checkbox checked={payCards} onChange={e => setPayCards(e.target.checked)} disabled={busy || (yellowLeft === 0 && redLeft === 0)}/>} label={yellowLeft || redLeft ? "Card fines" : "Card fines (none unpaid)"}/>
            {payCards && <div className="pay-cards">
              <Stepper label="Yellow cards" color="yellow" value={yellow} max={yellowLeft} onChange={setYellow} disabled={busy} price={yellowTotal}/>
              <Stepper label="Red cards" color="red" value={red} max={redLeft} onChange={setRed} disabled={busy} price={redTotal}/>
            </div>}
          </>}
        </section>

        {memberID && hasItems && <section className="pay-step">
          <h3><span>3</span> Upload your receipt</h3>
          <div className="pay-total"><span>Total for what you selected</span><strong>{money(total)}</strong></div>
          <TextField label="Amount you sent" value={amountSent} onChange={e => { setSentTouched(true); setAmountSent(e.target.value); }} required disabled={busy} inputMode="decimal"
            slotProps={{ input: { startAdornment: <InputAdornment position="start">₦</InputAdornment> } }}
            helperText={amountSent && Math.round(Number(amountSent.replace(/,/g, "")) * 100) !== total ? "This differs from the total. That’s fine, the admin will check it against your receipt." : " "}/>
          <TextField label="Date of transfer" type="date" value={paidOn} onChange={e => setPaidOn(e.target.value)} required disabled={busy} slotProps={{ inputLabel: { shrink: true }, htmlInput: { max: watDate() } }}/>
          <label className={`pay-file${file ? " has-file" : ""}`}>
            <input type="file" accept="image/jpeg,image/png,image/webp,application/pdf" onChange={e => setFile(e.target.files?.[0] ?? null)} disabled={busy} required/>
            <span>{file ? <><strong>{file.name}</strong> · {(file.size / 1024 / 1024).toFixed(1)} MB · tap to change</> : <><strong>Choose receipt</strong> · photo, screenshot or PDF, up to 5 MB</>}</span>
          </label>
          <TextField label="Note (optional)" value={note} onChange={e => setNote(e.target.value)} disabled={busy} multiline minRows={2} slotProps={{ htmlInput: { maxLength: 300 } }}/>
        </section>}
      </div></DialogContent>
      <DialogActions>
        <Button onClick={onClose} disabled={busy}>Cancel</Button>
        <Button type="submit" variant="contained" disabled={busy || !memberID || !hasItems || !file || !account}>{busy ? "Uploading…" : "Send receipt"}</Button>
      </DialogActions>
    </form>
  </Dialog>;
}

function Stepper({ label, color, value, max, onChange, disabled, price }: { label: string; color: "yellow" | "red"; value: number; max: number; onChange: (n: number) => void; disabled: boolean; price: number | null }) {
  return <div className={`pay-stepper ${color}`}>
    <span className="pay-stepper-label"><i aria-hidden="true"/>{label}<small>{max} unpaid{value > 0 && price !== null ? ` · ${money(price)}` : ""}</small></span>
    <span className="pay-stepper-controls">
      <IconButton size="small" aria-label={`Fewer ${label.toLowerCase()}`} onClick={() => onChange(Math.max(0, value - 1))} disabled={disabled || value === 0}><RemoveRounded/></IconButton>
      <output aria-live="polite" aria-label={label}>{value}</output>
      <IconButton size="small" aria-label={`More ${label.toLowerCase()}`} onClick={() => onChange(Math.min(max, value + 1))} disabled={disabled || value >= max}><AddRounded/></IconButton>
    </span>
  </div>;
}
