import { useEffect, useState, type FormEvent } from "react";
import { Button, CircularProgress, TextField } from "@mui/material";
import { ApiError, message, verifyAdminKey } from "../api";

export default function AdminLogin({ onVerified }: { onVerified: (key: string) => void }) {
  const [key, setKey] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  useEffect(() => {
    if (!error) return;
    const timer = window.setTimeout(() => setError(""), 4000);
    return () => window.clearTimeout(timer);
  }, [error]);
  const submit = async (event: FormEvent) => {
    event.preventDefault();
    if (busy) return;
    setError("");
    const candidate = key.trim();
    if (!candidate) { setError("Enter your admin key."); return; }
    setBusy(true);
    try {
      await verifyAdminKey(candidate);
      setKey("");
      onVerified(candidate);
    } catch (err) {
      setError(err instanceof ApiError && err.status === 401 ? "Incorrect admin key." : message(err));
    } finally { setBusy(false); }
  };
  return <main className="viewer-welcome-page">
    <a className="viewer-welcome-brand" href="/"><img src="/crest.svg" alt="" width="52" height="52"/><span>GreatMinds<b>Sport</b></span></a>
    <section className="viewer-access-card" aria-labelledby="admin-login-title">
      <div className="viewer-access-masthead"><span><i aria-hidden="true"/> ADMIN ACCESS</span><span>GMS</span></div>
      <div className="viewer-access-content">
        <span className="viewer-welcome-kicker">GREATMINDSSPORT</span>
        <h1 id="admin-login-title">Admin<br/><span>login</span></h1>
        <p className="viewer-welcome-description">Manage your club, members and matches.</p>
        <p id="admin-login-message" className={error?"admin-login-error viewer-admin-message":"viewer-admin-message"} aria-live="polite" aria-atomic="true">{error||"Enter your admin key for administrative access."}</p>
        <form onSubmit={submit} className="viewer-access-form admin-access-form">
          <TextField label="Admin key" type="password" value={key} onChange={e=>setKey(e.target.value)} required autoFocus autoComplete="off" disabled={busy} error={!!error} slotProps={{htmlInput:{"aria-describedby":"admin-login-message"}}} fullWidth/>
          <Button className="viewer-enter-button" type="submit" variant="contained" disabled={busy||!key.trim()} startIcon={busy?<CircularProgress size={18} color="inherit"/>:undefined}>{busy?"Checking…":<>Log in <span aria-hidden="true">↗</span></>}</Button>
        </form>
      </div>
      <div className="viewer-access-footer"><span>YOUR CLUB. ONE COMMUNITY.</span><a href="/">Back to GreatMindsSport <span aria-hidden="true">↗</span></a></div>
    </section>
  </main>;
}
