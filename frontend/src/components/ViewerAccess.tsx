import { useEffect, useState, type FormEvent, type ReactNode } from "react";
import { Alert, Button, CircularProgress, TextField } from "@mui/material";
import { api, ApiError, message } from "../api";
import { createViewerIdleTimer, VIEWER_IDLE_MS } from "../viewerSession";
import { isAdminPath } from "../navigation";

const VIEWER_LOGIN_REQUIRED=false;
export function ViewerGate({children}:{children:ReactNode}) {
 if(!VIEWER_LOGIN_REQUIRED)return <>{children}</>;
 const admin=isAdminPath(window.location.pathname);
 const [state,setState]=useState<"checking"|"locked"|"open"|"error">("checking");
 const [error,setError]=useState("");
 const [remaining,setRemaining]=useState(VIEWER_IDLE_MS);
 const check=async()=>{
  setState("checking");setError("");
  try {const session=await api<{idle_remaining_seconds:number}>("/viewer/session");setRemaining(session.idle_remaining_seconds*1000);setState("open");}
  catch(e){if(e instanceof ApiError&&e.status===401)setState("locked");else {setError(message(e));setState("error");}}
 };
 useEffect(()=>{
  if(admin)return;
  let active=true;
  void api<{idle_remaining_seconds:number}>("/viewer/session").then(session=>{if(active){setRemaining(session.idle_remaining_seconds*1000);setState("open");}}).catch(e=>{if(!active)return;if(e instanceof ApiError&&e.status===401)setState("locked");else {setError(message(e));setState("error");}});
  const lock=()=>{setError("");setState("locked");};
  window.addEventListener("viewer-access-required",lock);
  return()=>{active=false;window.removeEventListener("viewer-access-required",lock);};
 },[admin]);
 useEffect(()=>{
  if(admin||state!=="open")return;
  const channel=typeof BroadcastChannel!=="undefined"?new BroadcastChannel("greatminds-viewer-activity"):null;
  const timer=createViewerIdleTimer({remaining,onIdle:()=>{setState("locked");channel?.postMessage("logout");void api("/viewer/logout",{method:"POST"}).catch(()=>{});},onActivity:()=>{void api("/viewer/activity",{method:"POST",body:"{}"}).catch(()=>{});}});
  const activity=(event:Event)=>{if(!event.isTrusted||document.visibilityState!=="visible")return;if(timer.activity())channel?.postMessage("activity");};
  const check=()=>{timer.check();};
  if(channel)channel.onmessage=event=>{if(event.data==="activity")timer.activity(false);else if(event.data==="logout")setState("locked");};
  const events=["pointerdown","keydown","scroll","touchstart"];
  events.forEach(name=>document.addEventListener(name,activity,{passive:true,capture:true}));
  document.addEventListener("visibilitychange",check);window.addEventListener("focus",check);
  return()=>{timer.stop();channel?.close();events.forEach(name=>document.removeEventListener(name,activity,true));document.removeEventListener("visibilitychange",check);window.removeEventListener("focus",check);};
 },[admin,state,remaining]);
 if(admin||state==="open")return children;
 if(state==="checking")return <main className="admin-login-page"><div className="loading" role="status"><CircularProgress size={24}/><span>Checking access…</span></div></main>;
 if(state==="error")return <main className="admin-login-page"><Alert severity="error" action={<Button onClick={()=>void check()}>Retry</Button>}>{error}</Alert></main>;
 return <ViewerLogin onVerified={()=>{setRemaining(VIEWER_IDLE_MS);setState("open");}}/>;
}
function ViewerLogin({onVerified}:{onVerified:()=>void}) {
 const [pin,setPin]=useState(""),[busy,setBusy]=useState(false),[error,setError]=useState("");
 const submit=async(e:FormEvent)=>{
  e.preventDefault();if(busy||!/^(?=.*[A-Za-z])(?=.*[0-9])[A-Za-z0-9]{6}$/.test(pin))return;setBusy(true);setError("");
  try {await api("/viewer/login",{method:"POST",body:JSON.stringify({pin})});setPin("");onVerified();}
  catch(e){setError(message(e));setPin("");}finally{setBusy(false);}
 };
 return <main className="viewer-welcome-page">
  <div className="viewer-welcome-brand"><img src="/crest.svg" alt="" width="52" height="52"/><span>GreatMinds<b>Sport</b></span></div>
  <section className="viewer-access-card" aria-labelledby="viewer-login-title">
   <div className="viewer-access-masthead"><span><i aria-hidden="true"/> CLUB ACCESS</span><span>GMS</span></div>
   <div className="viewer-access-content">
    <span className="viewer-welcome-kicker">THE CLUB STARTS HERE</span>
    <h1 id="viewer-login-title">Enter your<br/><span>PIN</span></h1>
    <p className="viewer-welcome-description">Your people. Your game. Your club.</p>
    <div className="viewer-access-destinations" aria-label="Included sections"><span>Roaster</span><span>League</span></div>
    <form onSubmit={submit} className="viewer-access-form">
     <label className="viewer-pin-label" htmlFor="viewer-pin-input">Enter your six-character alphanumeric PIN</label>
     <div className={`viewer-pin-entry${error?" has-error":""}${busy?" is-busy":""}`}>
      <input id="viewer-pin-input" className="viewer-pin-input" aria-label="Viewer PIN" aria-describedby="viewer-login-message" aria-invalid={!!error} type="password" value={pin} onChange={e=>{setPin(e.target.value.replace(/[^A-Za-z0-9]/g,"").slice(0,6));setError("");}} autoFocus required disabled={busy} autoComplete="off" inputMode="text" pattern="(?=.*[A-Za-z])(?=.*[0-9])[A-Za-z0-9]{6}" maxLength={6} autoCapitalize="none" spellCheck={false}/>
      <div className="viewer-pin-slots" aria-hidden="true">{[0,1,2,3,4,5].map(i=><span key={i} className={`${pin.length>i?"is-filled":""}${pin.length===i?" is-next":""}`}>{pin.length>i?"●":<i/>}</span>)}</div>
     </div>
     <p id="viewer-login-message" className={`viewer-pin-feedback${error?" is-error":""}`} role={error?"alert":undefined}>{error||"Use the shared code from your admin. Letter case matters."}</p>
     <Button className="viewer-enter-button" type="submit" variant="contained" disabled={busy||pin.length!==6}>{busy?<><CircularProgress size={18} color="inherit"/> Checking PIN…</>:<>Enter GreatMindsSport <span aria-hidden="true">↗</span></>}</Button>
    </form>
   </div>
   <div className="viewer-access-footer"><span>YOUR CLUB. ONE COMMUNITY.</span><a href="/admin">Admin access <span aria-hidden="true">↗</span></a></div>
  </section>
  <p className="viewer-idle-note">Make yourself at home. You’ll be signed out after 2 hours of inactivity.</p>
 </main>;
}
export function ViewerPINSettings({token}:{token:string}) {
 const [pin,setPin]=useState(""),[confirm,setConfirm]=useState(""),[version,setVersion]=useState<number|null>(null),[busy,setBusy]=useState(false),[error,setError]=useState(""),[saved,setSaved]=useState(false);
 const load=async()=>{const result=await api<{version:number}>("/admin/viewer-pin",{},token);setVersion(result.version);};
 useEffect(()=>{void load().catch(e=>setError(message(e)));},[token]);
 const submit=async(e:FormEvent)=>{
  e.preventDefault();if(busy||version===null)return;setSaved(false);setError("");
  if(!/^(?=.*[A-Za-z])(?=.*[0-9])[A-Za-z0-9]{6}$/.test(pin)){setError("Enter exactly six characters with at least one letter and one number. Letter case matters.");return;}
  if(pin!==confirm){setError("The PINs do not match.");return;}
  setBusy(true);
  try{await api("/admin/viewer-pin",{method:"PUT",body:JSON.stringify({pin,version})},token);setPin("");setConfirm("");setSaved(true);await load();}
  catch(e){setError(message(e));if(e instanceof ApiError&&e.status===409)await load().catch(()=>{});}finally{setBusy(false);}
 };
 return <section className="panel"><h2>Viewer access</h2><p>One shared six-character alphanumeric PIN protects Roaster and League. Changing it signs out all viewers. Admin access remains separate.</p>
  {error&&<Alert severity="error">{error}</Alert>}{saved&&<Alert severity="success">Viewer PIN updated. Share the new PIN with members.</Alert>}
  <form onSubmit={submit} className="dialog-form">
   <TextField label="New viewer PIN" type="password" value={pin} onChange={e=>{setPin(e.target.value.replace(/[^A-Za-z0-9]/g,"").slice(0,6));setSaved(false);}} disabled={busy} required autoComplete="new-password" slotProps={{htmlInput:{inputMode:"text",maxLength:6,pattern:"(?=.*[A-Za-z])(?=.*[0-9])[A-Za-z0-9]{6}",autoCapitalize:"none",spellCheck:false}}}/>
   <TextField label="Confirm viewer PIN" type="password" value={confirm} onChange={e=>setConfirm(e.target.value.replace(/[^A-Za-z0-9]/g,"").slice(0,6))} disabled={busy} required autoComplete="new-password" slotProps={{htmlInput:{inputMode:"text",maxLength:6,pattern:"(?=.*[A-Za-z])(?=.*[0-9])[A-Za-z0-9]{6}",autoCapitalize:"none",spellCheck:false}}}/>
   <div><Button type="submit" variant="contained" disabled={busy||version===null||pin.length!==6||confirm.length!==6}>{busy?"Saving…":"Update viewer PIN"}</Button>{version===null&&<Button onClick={()=>void load().then(()=>setError("")).catch(e=>setError(message(e)))}>Retry</Button>}</div>
  </form>
 </section>;
}
export async function logoutViewer(){
 await api("/viewer/logout",{method:"POST"});
 if(typeof BroadcastChannel!=="undefined"){
  const channel=new BroadcastChannel("greatminds-viewer-activity");
  channel.postMessage("logout");channel.close();
 }
 window.dispatchEvent(new Event("viewer-access-required"));
}
