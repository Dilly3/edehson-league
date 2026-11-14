import { useEffect, useState, type FormEvent } from "react";
import { Alert, Button, Checkbox, CircularProgress, Dialog, FormControlLabel, DialogActions, DialogContent, DialogTitle, TextField } from "@mui/material";
import { createAdmin, deleteAdmin, listAdmins, message, updateAdmin, type AdminPermissions, type AdminUser } from "../api";

const USERNAME_RE = /^[a-z0-9._-]{3,32}$/;
export const passwordOK = (p: string) => p.length >= 8 && p.length <= 72 && /[A-Za-z]/.test(p) && /[0-9]/.test(p);
const when = (iso: string | null) => iso ? new Date(iso).toLocaleString(undefined, { dateStyle: "medium", timeStyle: "short" }) : "Never";

/** Super admin only: create, disable, reset and remove admin accounts. */
export default function AdminUsers({ token }: { token: string }) {
  const [admins, setAdmins] = useState<AdminUser[] | null>(null);
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [perms, setPerms] = useState<AdminPermissions>({ league: true, roaster: true });
  const [busy, setBusy] = useState(false);
  const [resetting, setResetting] = useState<AdminUser | null>(null);
  const [removing, setRemoving] = useState<AdminUser | null>(null);

  const load = async () => { setAdmins(await listAdmins(token)); };
  useEffect(() => { void load().catch(e => setError(message(e))); }, [token]);

  const run = async (work: () => Promise<unknown>, done: string) => {
    setBusy(true); setError(""); setNotice("");
    try { await work(); setNotice(done); await load(); return true; }
    catch (e) { setError(message(e)); return false; }
    finally { setBusy(false); }
  };

  const name = username.trim().toLowerCase();
  const create = async (e: FormEvent) => {
    e.preventDefault();
    if (!USERNAME_RE.test(name)) { setError("Usernames are 3–32 characters: lowercase letters, numbers, dots, dashes or underscores."); return; }
    if (!passwordOK(password)) { setError("Passwords need 8–72 characters with at least one letter and one number."); return; }
    if (!perms.league && !perms.roaster) { setError("Give the admin League permission, Roaster permission, or both."); return; }
    if (await run(() => createAdmin(token, name, password, perms), `Admin “${name}” created. Share the username and password with them privately.`)) { setUsername(""); setPassword(""); setPerms({ league: true, roaster: true }); }
  };

  return <section className="panel admin-users">
    <h2>Admins</h2>
    <p>You are the super admin. Give each admin <strong>League</strong> (teams, players, fixtures, results), <strong>Roaster</strong> (members, dues, cards, donations, fees), or both. Only you can add or remove admins.</p>
    {error && <Alert severity="error" onClose={() => setError("")}>{error}</Alert>}
    {notice && <Alert severity="success" onClose={() => setNotice("")}>{notice}</Alert>}

    <form className="admin-users-form" onSubmit={create}>
      <TextField label="Username" value={username} onChange={e => setUsername(e.target.value.toLowerCase())} disabled={busy} required autoComplete="off" slotProps={{ htmlInput: { autoCapitalize: "none", spellCheck: false, maxLength: 32 } }}/>
      <TextField label="Password" type="password" value={password} onChange={e => setPassword(e.target.value)} disabled={busy} required autoComplete="new-password" helperText="8+ characters, with a letter and a number. Not case-sensitive." slotProps={{ htmlInput: { maxLength: 72 } }}/>
      <fieldset className="admin-perms">
        <legend>Permissions</legend>
        <FormControlLabel control={<Checkbox checked={perms.league} onChange={e => setPerms(p => ({ ...p, league: e.target.checked }))} disabled={busy}/>} label="League"/>
        <FormControlLabel control={<Checkbox checked={perms.roaster} onChange={e => setPerms(p => ({ ...p, roaster: e.target.checked }))} disabled={busy}/>} label="Roaster"/>
      </fieldset>
      <Button type="submit" variant="contained" disabled={busy || !name || !password || (!perms.league && !perms.roaster)}>Create admin</Button>
    </form>

    {admins === null ? <div className="loading" role="status"><CircularProgress size={20}/><span>Loading admins…</span></div>
      : admins.length === 0 ? <p className="membership-note">No admins yet. Create one above.</p>
      : <ul className="admin-users-list">
        {admins.map(a => <li key={a.id} className={a.active ? "" : "inactive"}>
          <div>
            <strong>{a.username}</strong>
            <small>{a.active ? "Active" : "Disabled"} · Last login: {when(a.last_login_at)}</small>
            <span className="admin-perm-chips" role="group" aria-label={`Permissions for ${a.username}`}>
              {([["can_league", "League"], ["can_roaster", "Roaster"]] as const).map(([field, label]) => {
                const on = a[field], other = field === "can_league" ? a.can_roaster : a.can_league;
                return <button key={field} type="button" className={on ? "on" : ""} aria-pressed={on} disabled={busy || (on && !other)}
                  title={on && !other ? "An admin needs at least one permission" : on ? `Remove ${label} permission` : `Give ${label} permission`}
                  onClick={() => void run(() => updateAdmin(token, a.id, { [field]: !on }), `${label} permission ${on ? "removed from" : "given to"} “${a.username}”.`)}>
                  {on ? "✓ " : "+ "}{label}
                </button>;
              })}
            </span>
          </div>
          <div className="membership-buttons">
            <Button size="small" disabled={busy} onClick={() => setResetting(a)}>Reset password</Button>
            <Button size="small" disabled={busy} onClick={() => void run(() => updateAdmin(token, a.id, { active: !a.active }), a.active ? `“${a.username}” disabled and signed out.` : `“${a.username}” enabled.`)}>{a.active ? "Disable" : "Enable"}</Button>
            <Button size="small" color="error" disabled={busy} onClick={() => setRemoving(a)}>Remove</Button>
          </div>
        </li>)}
      </ul>}

    {resetting && <ResetPassword admin={resetting} busy={busy} onClose={() => setResetting(null)}
      onSave={async pw => { if (await run(() => updateAdmin(token, resetting.id, { password: pw }), `Password for “${resetting.username}” changed. They have been signed out.`)) setResetting(null); }}/>}

    <Dialog open={!!removing} onClose={() => setRemoving(null)} maxWidth="xs">
      <DialogTitle>Remove admin?</DialogTitle>
      <DialogContent><p>“{removing?.username}” will be signed out and can no longer log in. This can’t be undone, but you can create the account again.</p></DialogContent>
      <DialogActions>
        <Button onClick={() => setRemoving(null)}>Cancel</Button>
        <Button color="error" variant="contained" disabled={busy} onClick={async () => { const r = removing!; if (await run(() => deleteAdmin(token, r.id), `“${r.username}” removed.`)) setRemoving(null); }}>Remove</Button>
      </DialogActions>
    </Dialog>
  </section>;
}

function ResetPassword({ admin, busy, onClose, onSave }: { admin: AdminUser; busy: boolean; onClose: () => void; onSave: (pw: string) => void }) {
  const [pw, setPw] = useState(""), [confirm, setConfirm] = useState(""), [err, setErr] = useState("");
  const submit = (e: FormEvent) => {
    e.preventDefault();
    if (!passwordOK(pw)) { setErr("Use 8–72 characters with at least one letter and one number."); return; }
    if (pw !== confirm) { setErr("The passwords do not match."); return; }
    onSave(pw);
  };
  return <Dialog open onClose={onClose} maxWidth="xs">
    <form onSubmit={submit}>
      <DialogTitle>Reset password</DialogTitle>
      <DialogContent className="dialog-form">
        <p>Set a new password for “{admin.username}”. They’ll be signed out everywhere.</p>
        {err && <Alert severity="error">{err}</Alert>}
        <TextField label="New password" type="password" value={pw} onChange={e => setPw(e.target.value)} autoComplete="new-password" required autoFocus/>
        <TextField label="Confirm password" type="password" value={confirm} onChange={e => setConfirm(e.target.value)} autoComplete="new-password" required/>
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>Cancel</Button>
        <Button type="submit" variant="contained" disabled={busy || !pw || !confirm}>Save password</Button>
      </DialogActions>
    </form>
  </Dialog>;
}
