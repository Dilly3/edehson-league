import ScrollTable from "./ScrollTable";
import { Button } from "@mui/material";
import { individualCardPayments, money, type Card, type Payment, type Member, type RegisterData } from "../membership";

export default function MemberCardPayments({data,members,color,year,month,onEdit,onUndo,onEditCard,showPlayer=true}:{data:RegisterData;members:Member[];color:"yellow"|"red";year:number;month:string;showPlayer?:boolean;onUndo?:(payment:Payment,member:Member)=>void;onEdit?:(payment:Payment,member:Member)=>void;onEditCard?:(card:Card,index:number,member:Member)=>void}) {
 const memberMap=new Map(members.map(m=>[m.id,m]));
 const cards=data.cards.filter(c=>!c.void&&c.color===color&&memberMap.has(c.member_id)&&(c.occurred_on?c.occurred_on.startsWith(`${year}-${month==="all"?"":month}`):month==="all"));
 const totals=cards.reduce((sum,card)=>{
  for(const row of individualCardPayments(data,card)) {
   if(row.payment){sum.paid+=card.unit_amount;sum.settled++;}
   else {sum.outstanding+=card.unit_amount;sum.unpaid++;}
  }
  return sum;
 },{paid:0,outstanding:0,settled:0,unpaid:0});
 return <section aria-label={`${color} card payments`}>
  <dl className="member-payment-totals"><div><dt>{color==="yellow"?"Yellow":"Red"} card fines paid</dt><dd>{money(totals.paid)} <small>({totals.settled} cards)</small></dd></div><div><dt>{color==="yellow"?"Yellow":"Red"} card fines outstanding</dt><dd>{money(totals.outstanding)} <small>({totals.unpaid} cards)</small></dd></div></dl>
  <p className="membership-note">Totals reflect the cards matching the selected player, issue year and month, including their linked payments. </p>
  <p className="membership-note">One row per card, filtered by issue date. Undated historical cards appear under All months regardless of year. Historical grouped settlements are assigned to rows in a consistent order; the source did not identify individual cards.</p>
  <ScrollTable className="membership-table-scroll"><table className="membership-table"><caption className="visually-hidden">{color} cards and linked payments</caption><thead><tr>{showPlayer&&<th scope="col">Player</th>}<th scope="col">Card Type</th><th scope="col">Issue date</th><th scope="col">Fee</th><th scope="col">Status</th><th scope="col">Payment date</th>{(onEdit||onEditCard||onUndo)&&<th scope="col">Manage</th>}</tr></thead><tbody>{cards.flatMap(c=>individualCardPayments(data,c).map(({index,payment})=>{
   const member=memberMap.get(c.member_id)!;
   return <tr key={`${c.id}-${index}`} className={member.active?undefined:"membership-row-inactive"}>{showPlayer&&<th scope="row" className="member-name-cell">{member.name}</th>}<td>{color==="yellow"?"Yellow":"Red"}</td><td>{c.occurred_on||"Not recorded"}</td><td>{money(c.unit_amount)}</td><td>{payment?"✅":"⌛️"}</td><td>{payment?(payment.paid_on||"Not recorded"):"—"}</td>{(onEdit||onEditCard||onUndo)&&<td><div className="membership-buttons">{onEditCard&&<Button size="small" onClick={()=>onEditCard(c,index,member)}>Edit card date</Button>}{onEdit&&payment&&<Button size="small" onClick={()=>onEdit(payment,member)}>Edit payment date</Button>}{onUndo&&payment&&<Button size="small" color="warning" onClick={()=>onUndo(payment,member)}>Undo payment</Button>}</div></td>}</tr>;
  }))}</tbody></table></ScrollTable>
  {!cards.length&&<p className="membership-note">No {color} card records match these filters.</p>}
 </section>;
}
