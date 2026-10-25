import { useState } from "react";
import { money } from "../membership";

export default function MemberFineCard({color,summary}:{color:"yellow"|"red";summary:{received:number;paid:number;outstanding:number;balance:number}}){
 const [flipped,setFlipped]=useState(false);
 const name=color==="yellow"?"Yellow":"Red";
 return <button type="button" className={`member-fine-card member-fine-card-${color}${flipped?" is-flipped":""}`} onClick={()=>setFlipped(value=>!value)} aria-pressed={flipped} aria-label={flipped?`${name} cards: ${summary.received} received, ${summary.paid} paid, ${summary.outstanding} outstanding, balance ${money(summary.balance)}. Click to turn back.`:`Show ${name.toLowerCase()} card statistics`}>
  <span className="member-fine-card-inner">
   <span className="member-fine-card-front" aria-hidden={flipped}>
    <strong>{name}<br/>cards</strong>
    <span>Tap to view stats ↻</span>
   </span>
   <span className="member-fine-card-back" aria-hidden={!flipped}>
    <strong>{name} cards</strong>
    <span className="member-fine-card-stats">
     <span><span>Received</span><b>{summary.received}</b></span>
     <span><span>Paid</span><b>{summary.paid}</b></span>
     <span><span>Outstanding</span><b>{summary.outstanding}</b></span>
    </span>
    <span className="member-fine-card-balance"><span>Balance</span><strong>{money(summary.balance)}</strong></span>
    <span className="member-fine-card-hint">Tap to turn back ↻</span>
   </span>
  </span>
 </button>;
}
