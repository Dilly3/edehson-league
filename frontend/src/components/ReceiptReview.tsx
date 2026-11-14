import { useEffect, useState } from "react";
import { Alert, Button, CircularProgress, Dialog, DialogActions, DialogContent, DialogTitle, MenuItem, Tab, Tabs, TextField } from "@mui/material";
import { message } from "../api";
import { duesMonthLabel, money } from "../membership";
import { confirmReceipt, deleteReceiptFile, getReceiptSettings, listReceipts, purgeReceiptFiles, receiptsCSV, receiptURL, rejectReceipt, retentionLabel, saveReceiptSettings, type PaymentRequest } from "../payments";

const shortMonth = (m: string) => duesMonthLabel(m).replace("-", " ");
const when = (iso: string) => new Date(iso).toLocaleString("en-NG", { timeZone: "Africa/Lagos", dateStyle: "medium", timeStyle: "short" });

/** Roaster admin: check uploaded receipts against what the player says they paid for, then confirm or reject. */
export default function ReceiptReview({ token, superAdmin, onClose, onChanged }: { token: string; superAdmin: boolean; onClose: () => void; onChanged: () => void }) {
  const [tab, setTab] = useState<"pending" | "all">("pending");
  const [statusFilter, setStatusFilter] = useState<"any" | "confirmed" | "rejected" | "pending">("any");
  const [query, setQuery] = useState("");
  const [purgeOpen, setPurgeOpen] = useState(false);
  const [purgeDays, setPurgeDays] = useState("90");
  const [retention, setRetention] = useState<number | null>(null);
  const [editingRetention, setEditingRetention] = useState(false);
  const [retentionInput, setRetentionInput] = useState("");
  useEffect(() => { void getReceiptSettings(token).then(s => { setRetention(s.retention_days); setPurgeDays(String(s.retention_days)); }).catch(() => {}); }, [token]);
  const saveRetention = async () => {
    const days = Number(retentionInput);
    if (!Number.isInteger(days) || days < 1 || days > 3650) { setError("Keep receipts for between 1 and 3650 days."); return; }
    setBusy("retention"); setError(""); setNotice("");
    try { const s = await saveReceiptSettings(token, days); setRetention(s.retention_days); setEditingRetention(false); setNotice(`Receipt files will now be kept for ${retentionLabel(s.retention_days)} after review.`); }
    catch (e) { setError(message(e)); }
    finally { setBusy(""); }
  };
  const [items, setItems] = useState<PaymentRequest[] | null>(null);
  const [error, setError] = useState(""), [notice, setNotice] = useState("");
  const [busy, setBusy] = useState("");
  const [preview, setPreview] = useState<{ url: string; type: string; name: string } | null>(null);
  const [rejecting, setRejecting] = useState<PaymentRequest | null>(null);
  const [reason, setReason] = useState("");

  const load = async () => { setItems(null); setItems(await listReceipts(token, tab)); };
  useEffect(() => { void load().catch(e => { setItems([]); setError(message(e)); }); }, [tab, token]);
  useEffect(() => () => { if (preview) URL.revokeObjectURL(preview.url); }, [preview]);

  const view = async (r: PaymentRequest) => {
    setBusy(r.id); setError("");
    try { setPreview({ url: await receiptURL(token, r.id), type: r.receipt_type, name: r.receipt_name }); }
    catch (e) { setError(message(e)); }
    finally { setBusy(""); }
  };
  const act = async (id: string, work: () => Promise<unknown>, done: string) => {
    setBusy(id); setError(""); setNotice("");
    try { await work(); setNotice(done); onChanged(); await load(); return true; }
    catch (e) { setError(message(e)); return false; }
    finally { setBusy(""); }
  };

  const shown = (items ?? []).filter(r => tab === "pending" || ((statusFilter === "any" || r.status === statusFilter) && r.member_name.toLowerCase().includes(query.trim().toLowerCase())));
  const download = () => {
    const blob = new Blob([receiptsCSV(shown)], { type: "text/csv;charset=utf-8" });
    const a = document.createElement("a");
    a.href = URL.createObjectURL(blob);
    a.download = `payment-receipts-${new Date().toISOString().slice(0, 10)}.csv`;
    a.click();
    setTimeout(() => URL.revokeObjectURL(a.href), 1000);
  };

  return <Dialog open onClose={onClose} maxWidth="md" fullWidth className="receipt-review">
    <DialogTitle>Payment receipts</DialogTitle>
    <DialogContent><div className="dialog-form">
      <Tabs value={tab} onChange={(_, v) => setTab(v)}><Tab value="pending" label="Awaiting review"/><Tab value="all" label="History"/></Tabs>
      {tab === "all" && <div className="receipt-history-tools">
        <TextField label="Player" size="small" value={query} onChange={e => setQuery(e.target.value)} placeholder="Search by name"/>
        <TextField select label="Status" size="small" value={statusFilter} onChange={e => setStatusFilter(e.target.value as typeof statusFilter)}>
          <MenuItem value="any">All statuses</MenuItem><MenuItem value="confirmed">Confirmed</MenuItem><MenuItem value="rejected">Rejected</MenuItem><MenuItem value="pending">Awaiting review</MenuItem>
        </TextField>
        <Button onClick={download} disabled={!shown.length}>Download CSV</Button>
        {superAdmin && <Button color="error" onClick={() => setPurgeOpen(true)}>Delete old files</Button>}
      </div>}
      {tab === "all" && <div className="receipt-retention">
        {editingRetention ? <form className="receipt-retention-form" onSubmit={e => { e.preventDefault(); void saveRetention(); }}>
          <TextField label="Keep reviewed receipt files for (days)" type="number" size="small" value={retentionInput} onChange={e => setRetentionInput(e.target.value)} autoFocus
            slotProps={{ htmlInput: { min: 1, max: 3650 } }} helperText="Checked once a week. Waiting receipts are never deleted."/>
          <Button type="submit" variant="contained" disabled={busy === "retention"}>Save</Button>
          <Button onClick={() => setEditingRetention(false)}>Cancel</Button>
        </form> : <p className="muted">
          Receipt files are kept for {retention === null ? "…" : retentionLabel(retention)} after review, then deleted in the weekly cleanup. The records below stay, so download the CSV for your books.
          {superAdmin && retention !== null && <Button size="small" onClick={() => { setRetentionInput(String(retention)); setEditingRetention(true); }}>Change</Button>}
        </p>}
      </div>}
      {error && <Alert severity="error" onClose={() => setError("")}>{error}</Alert>}
      {notice && <Alert severity="success" onClose={() => setNotice("")}>{notice}</Alert>}
      {items === null ? <div className="loading" role="status"><CircularProgress size={20}/><span>Loading receipts…</span></div>
        : !shown.length ? <p className="membership-note">{tab === "pending" ? "No receipts are waiting for review." : items.length ? "No receipts match these filters." : "No receipts yet."}</p>
        : <ul className="receipt-list">{shown.map(r => {
          const tally = r.amount_sent === r.amount;
          return <li key={r.id} className={`receipt-item ${r.status}`}>
            <header>
              <strong className="receipt-player">{r.member_name}</strong>
              <span className={`receipt-status ${r.status}`}>{r.status === "pending" ? "Awaiting review" : r.status === "confirmed" ? "Confirmed" : "Rejected"}</span>
            </header>
            <small className="muted">Sent {when(r.created_at)} · transfer dated {r.paid_on}</small>
            <ul className="receipt-lines">
              {r.months.length > 0 && <li><span>Monthly dues · {r.months.map(shortMonth).join(", ")}</span><b>{money(r.dues_amount)}</b></li>}
              {r.yellow_count > 0 && <li><span><i className="dot yellow"/>Yellow cards × {r.yellow_count}</span><b>{money(r.yellow_amount)}</b></li>}
              {r.red_count > 0 && <li><span><i className="dot red"/>Red cards × {r.red_count}</span><b>{money(r.red_amount)}</b></li>}
              <li className="receipt-total"><span>Expected total</span><b>{money(r.amount)}</b></li>
              <li className={`receipt-sent ${tally ? "ok" : "mismatch"}`}><span>Player says they sent</span><b>{money(r.amount_sent)}</b></li>
            </ul>
            {!tally && <Alert severity="warning">The amount sent is {money(Math.abs(r.amount_sent - r.amount))} {r.amount_sent > r.amount ? "more" : "less"} than the expected total. Check the receipt before confirming.</Alert>}
            {r.note && <p className="receipt-note">“{r.note}”</p>}
            {r.status === "rejected" && r.review_note && <p className="receipt-note">Rejected: {r.review_note}</p>}
            {r.status !== "pending" && r.reviewed_at && <small className="muted receipt-reviewer">{r.status === "confirmed" ? "Confirmed" : "Rejected"} by {r.reviewed_by || "an admin"} · {when(r.reviewed_at)}</small>}
            <div className="membership-buttons">
              {r.receipt_deleted_at
                ? <span className="receipt-file-gone">Receipt file deleted {when(r.receipt_deleted_at)}</span>
                : <Button disabled={!!busy} onClick={() => void view(r)}>{busy === r.id && !rejecting ? "Opening…" : "View receipt"}</Button>}
              {superAdmin && r.status !== "pending" && !r.receipt_deleted_at && <Button color="error" disabled={!!busy}
                onClick={() => { if (window.confirm(`Delete the receipt file from ${r.member_name}? The record and payments stay.`)) void act(r.id, () => deleteReceiptFile(token, r.id), "Receipt file deleted."); }}>Delete file</Button>}
              {r.status === "pending" && <>
                <Button variant="contained" disabled={!!busy} onClick={() => void act(r.id, () => confirmReceipt(token, r.id), `Confirmed. ${r.member_name}’s record is updated.`)}>Confirm & record</Button>
                <Button color="error" disabled={!!busy} onClick={() => { setRejecting(r); setReason(""); }}>Reject</Button>
              </>}
            </div>
          </li>;
        })}</ul>}
    </div></DialogContent>
    <DialogActions><Button onClick={onClose}>Close</Button></DialogActions>

    <Dialog open={!!preview} onClose={() => setPreview(null)} maxWidth="md" fullWidth>
      <DialogTitle>Receipt</DialogTitle>
      <DialogContent>{preview && (preview.type === "application/pdf"
        ? <iframe className="receipt-preview" src={preview.url} title="Receipt PDF"/>
        : <img className="receipt-preview" src={preview.url} alt={`Receipt ${preview.name}`}/>)}</DialogContent>
      <DialogActions>{preview && <Button href={preview.url} target="_blank" rel="noopener">Open in new tab</Button>}<Button onClick={() => setPreview(null)}>Close</Button></DialogActions>
    </Dialog>

    <Dialog open={purgeOpen} onClose={() => setPurgeOpen(false)} maxWidth="xs" fullWidth>
      <DialogTitle>Delete old receipt files</DialogTitle>
      <DialogContent><div className="dialog-form">
        <p>Deletes the stored files for confirmed and rejected receipts reviewed more than this many days ago. Receipts waiting for review are never touched, and every record and payment stays.</p>
        <TextField label="Reviewed more than (days) ago" type="number" value={purgeDays} onChange={e => setPurgeDays(e.target.value)} slotProps={{ htmlInput: { min: 0, max: 3650 } }}/>
      </div></DialogContent>
      <DialogActions>
        <Button onClick={() => setPurgeOpen(false)}>Cancel</Button>
        <Button color="error" variant="contained" disabled={!!busy || !/^\d+$/.test(purgeDays)} onClick={async () => {
          setBusy("purge"); setError(""); setNotice("");
          try {
            const r = await purgeReceiptFiles(token, Number(purgeDays));
            setNotice(r.deleted ? `Deleted ${r.deleted} receipt file${r.deleted === 1 ? "" : "s"} (${(r.bytes_freed / 1024 / 1024).toFixed(1)} MB freed).${r.failed ? ` ${r.failed} could not be deleted; try again later.` : ""}` : "No receipt files were old enough to delete.");
            setPurgeOpen(false); await load();
          } catch (e) { setError(message(e)); } finally { setBusy(""); }
        }}>Delete files</Button>
      </DialogActions>
    </Dialog>

    <Dialog open={!!rejecting} onClose={() => setRejecting(null)} maxWidth="xs" fullWidth>
      <DialogTitle>Reject receipt?</DialogTitle>
      <DialogContent><div className="dialog-form">
        <p>Nothing is recorded for {rejecting?.member_name}. Tell them why so they can resend.</p>
        <TextField label="Reason" value={reason} onChange={e => setReason(e.target.value)} required autoFocus multiline minRows={2} slotProps={{ htmlInput: { maxLength: 300 } }}/>
      </div></DialogContent>
      <DialogActions>
        <Button onClick={() => setRejecting(null)}>Cancel</Button>
        <Button color="error" variant="contained" disabled={!reason.trim() || !!busy} onClick={async () => { const r = rejecting!; if (await act(r.id, () => rejectReceipt(token, r.id, reason.trim()), `Receipt from ${r.member_name} rejected.`)) setRejecting(null); }}>Reject</Button>
      </DialogActions>
    </Dialog>
  </Dialog>;
}
