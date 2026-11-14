import { useEffect, useState, type FormEvent } from "react";
import { Button, CircularProgress, TextField } from "@mui/material";
import { ApiError, loginAdmin, message, verifyAdminKey, type AdminSession } from "../api";

type Mode = "account" | "key";

export default function AdminLogin({ onVerified }: { onVerified: (session: AdminSession) => void }) {
  const [mode, setMode] = useState<Mode>("account");
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [key, setKey] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  useEffect(() => {
    if (!error) return;
    const timer = window.setTimeout(() => setError(""), 4000);
    return () => window.clearTimeout(timer);
  }, [error]);
  const switchMode = (next: Mode) => { setMode(next); setError(""); setPassword(""); setKey(""); };
  const ready = mode === "key" ? !!key.trim() : !!username.trim() && !!password;
  const submit = async (event: FormEvent) => {
    event.preventDefault();
    if (busy || !ready) return;
    setError("");
    setBusy(true);
    try {
      const session = mode === "key" ? await verifyAdminKey(key.trim()) : await loginAdmin(username.trim(), password);
      setKey(""); setPassword("");
      onVerified(session);
    } catch (err) {
      if (err instanceof ApiError && err.status === 401) setError(mode === "key" ? "Incorrect super admin key." : "Incorrect username or password.");
      else setError(message(err));
      setPassword("");
    } finally { setBusy(false); }
  };
  const hint = mode === "key" ? "Enter the super admin key to manage admins and the club." : "Sign in with the username and password your super admin gave you.";
  return <main className="viewer-welcome-page">
    <a className="viewer-welcome-brand" href="/"><img src="/crest.svg" alt="" width="52" height="52"/><span>GreatMinds<b>Sport</b></span></a>
    <section className="viewer-access-card" aria-labelledby="admin-login-title">
      <div className="viewer-access-masthead"><span><i aria-hidden="true"/> ADMIN ACCESS</span><span>GMS</span></div>
      <div className="viewer-access-content">
        <span className="viewer-welcome-kicker">GREATMINDSSPORT</span>
        <h1 id="admin-login-title">Admin<br/><span>login</span></h1>
        <p className="viewer-welcome-description">Manage your club, members and matches.</p>
        <div className="admin-login-modes" role="tablist" aria-label="Sign-in method">
          <button type="button" role="tab" aria-selected={mode === "account"} className={mode === "account" ? "active" : ""} onClick={() => switchMode("account")} disabled={busy}>Admin</button>
          <button type="button" role="tab" aria-selected={mode === "key"} className={mode === "key" ? "active" : ""} onClick={() => switchMode("key")} disabled={busy}>Super admin</button>
        </div>
        <p id="admin-login-message" className={error?"admin-login-error viewer-admin-message":"viewer-admin-message"} aria-live="polite" aria-atomic="true">{error||hint}</p>
        <form onSubmit={submit} className="viewer-access-form admin-access-form">
          {mode === "account" ? <>
            <TextField label="Username" value={username} onChange={e=>setUsername(e.target.value)} required autoFocus autoComplete="username" disabled={busy} error={!!error} slotProps={{htmlInput:{autoCapitalize:"none",spellCheck:false,"aria-describedby":"admin-login-message"}}} fullWidth/>
            <TextField label="Password" type="password" value={password} onChange={e=>setPassword(e.target.value)} required autoComplete="current-password" disabled={busy} error={!!error} slotProps={{htmlInput:{"aria-describedby":"admin-login-message"}}} fullWidth/>
          </> :
            <TextField label="Super admin key" type="password" value={key} onChange={e=>setKey(e.target.value)} required autoFocus autoComplete="off" disabled={busy} error={!!error} slotProps={{htmlInput:{"aria-describedby":"admin-login-message"}}} fullWidth/>}
          <Button className="viewer-enter-button" type="submit" variant="contained" disabled={busy||!ready} startIcon={busy?<CircularProgress size={18} color="inherit"/>:undefined}>{busy?"Checking…":<>Log in <span aria-hidden="true">↗</span></>}</Button>
        </form>
      </div>
      <div className="viewer-access-footer"><span>YOUR CLUB. ONE COMMUNITY.</span><a href="/">Back to GreatMindsSport <span aria-hidden="true">↗</span></a></div>
    </section>
  </main>;
}
