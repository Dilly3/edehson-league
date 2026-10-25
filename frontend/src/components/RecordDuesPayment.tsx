import { useRef, useState, type FormEvent } from "react";
import { Alert, Button, CircularProgress, Dialog, DialogActions, DialogContent, DialogTitle, MenuItem, TextField } from "@mui/material";
import { api, message } from "../api";
import { duesPaymentMinimum, duesMonthLabel, duesPaymentDate, money, watDate, type Member, type RegisterData } from "../membership";

export default function RecordDuesPayment({data,token,initialMember,year,onYearChange,loading,loadError,onReload,onClose,onSaved}:{
 data:RegisterData;token:string;initialMember?:Member;year:number;onYearChange:(year:number)=>void;
 loading:boolean;loadError:string;onReload:()=>void;onClose:()=>void;onSaved:()=>void;
}) {
 const [memberID,setMemberID]=useState(initialMember?.id??""),[dueID,setDueID]=useState("");
 const [date,setDate]=useState(""),[note,setNote]=useState(""),[busy,setBusy]=useState(false),[error,setError]=useState("");
 const retry=useRef<{body:string;id:string}|null>(null);
 const dues=data.dues.filter(d=>d.member_id===memberID&&d.month.startsWith(`${year}-`)&&d.status==="unpaid").sort((a,b)=>a.month.localeCompare(b.month));
 const due=dues.find(d=>d.id===dueID);
 const member=data.members.find(m=>m.id===memberID);
 const latestYear=Math.max(Number(watDate().slice(0,4)),year);
 const years=Array.from({length:latestYear-2026+1},(_,i)=>latestYear-i);
 const submit=async(e:FormEvent)=>{
  e.preventDefault();if(busy||loading||loadError)return;setError("");
  if(!member||!due){setError("Select a member and an outstanding month.");return;}
  if(!date||date<duesPaymentMinimum(due.month,member.joined_on)||date>watDate()){setError("Choose a payment date from the joining month, not in the future.");return;}
  const body={type:"pay_due",member_id:member.id,record_id:due.id,revision:due.revision,paid_on:date,note:note.trim()};
  const fingerprint=JSON.stringify(body);
  if(retry.current?.body!==fingerprint)retry.current={body:fingerprint,id:crypto.randomUUID()};
  setBusy(true);
  try{await api("/membership/commands",{method:"POST",body:JSON.stringify({...body,request_id:retry.current.id})},token);onSaved();}
  catch(e){setError(message(e));}finally{setBusy(false);}
 };
 return <Dialog open onClose={busy?undefined:onClose} maxWidth="sm" fullWidth><form onSubmit={submit}>
  <DialogTitle>Record dues payment</DialogTitle>
  <DialogContent><div className="dialog-form">
   {error&&<Alert severity="error" action={<Button disabled={busy} onClick={onReload}>Refresh dues</Button>}>{error}</Alert>}
   {loadError&&<Alert severity="error" action={<Button onClick={onReload}>Retry</Button>}>{loadError}</Alert>}
   <TextField select label="Member" value={memberID} onChange={e=>{setMemberID(e.target.value);setDueID("");setDate("");setError("");}} required disabled={busy}>
    <MenuItem value="" disabled>Select a member</MenuItem>{data.members.map(m=><MenuItem key={m.id} value={m.id}>{m.name}{m.active?"":" · Inactive"}</MenuItem>)}
   </TextField>
   <TextField select label="Year" value={year} onChange={e=>{setDueID("");setDate("");onYearChange(Number(e.target.value));}} disabled={busy||loading}>{years.map(y=><MenuItem key={y} value={y}>{y}</MenuItem>)}</TextField>
   {loading?<div className="loading" role="status"><CircularProgress size={20}/><span>Loading dues…</span></div>:<>
    <TextField select label="Month to pay" value={due?dueID:""} onChange={e=>{setDueID(e.target.value);const selected=dues.find(d=>d.id===e.target.value);setDate(selected?duesPaymentDate(selected.month):"");setError("");}} required disabled={busy||!memberID||!dues.length||!!loadError} helperText="Only issued, unpaid months are listed. Paid and waived months are excluded.">
     <MenuItem value="" disabled>Select an outstanding month</MenuItem>{dues.map(d=><MenuItem key={d.id} value={d.id}>{duesMonthLabel(d.month)} · {money(d.amount)}</MenuItem>)}
    </TextField>
    {memberID&&!dues.length&&!loadError&&<Alert severity="info">No outstanding monthly dues for {year}.</Alert>}
    {due&&<><Alert severity="info">Payment amount: <strong>{money(due.amount)}</strong>. This settles the selected month in full.</Alert>
     <TextField label="Payment date" type="date" value={date} onChange={e=>setDate(e.target.value)} required disabled={busy} slotProps={{inputLabel:{shrink:true},htmlInput:{min:duesPaymentMinimum(due.month,member?.joined_on??due.month+"-01"),max:watDate()}}}/>
    </>}
   </>}
   <TextField label="Note (optional)" value={note} onChange={e=>setNote(e.target.value)} disabled={busy} multiline minRows={2} slotProps={{htmlInput:{maxLength:300}}} helperText="Notes are publicly visible. Do not include private information."/>
  </div></DialogContent>
  <DialogActions><Button onClick={onClose} disabled={busy}>Cancel</Button><Button type="submit" variant="contained" disabled={busy||loading||!!loadError||!due||!date}>{busy?"Recording…":"Record payment"}</Button></DialogActions>
 </form></Dialog>;
}
