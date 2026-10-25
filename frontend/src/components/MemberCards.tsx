import { useState } from "react";
import { cardSummary, money, type RegisterData } from "../membership";

export default function MemberCards({data,memberID}:{data:RegisterData;memberID:string}) {
 const [flipped,setFlipped]=useState(false);
 return <button type="button" className={`member-fine-card member-combined-cards${flipped?" is-flipped":""}`} onClick={()=>setFlipped(value=>!value)} aria-pressed={flipped} aria-label={flipped?"Yellow and red card statistics. Tap to turn back.":"Show yellow and red card statistics"}>
  <span className="member-fine-card-inner">
   <span className="member-fine-card-front" aria-hidden={flipped}><strong>Cards</strong><span>Tap to view stats ↻</span></span>
   <span className="member-fine-card-back combined-cards-back" aria-hidden={!flipped}>
    {(["yellow","red"] as const).map(color=>{const stats=cardSummary(data,memberID,color);return <span className={`combined-card-section combined-${color}`} key={color}><strong>{color==="yellow"?"Yellow":"Red"} cards</strong><span>Received <b>{stats.received}</b> · Paid <b>{stats.paid}</b></span><span>Outstanding <b>{stats.outstanding}</b></span><span>{money(stats.balance)}</span></span>;})}
    <span className="member-fine-card-hint">Tap to turn back ↻</span>
   </span>
  </span>
 </button>;
}
