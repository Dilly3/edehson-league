import ScrollTable from "./ScrollTable";
import { Button } from "@mui/material";
import { money, type Payment, type Member, type RegisterData } from "../membership";

export default function MemberDonations({data,members,onSelect,onEdit,onUndo}:{data:RegisterData;members:Member[];onSelect?:(member:Member)=>void;onUndo?:(payment:Payment,member:Member)=>void;onEdit?:(payment:Payment,member:Member)=>void}) {
 const byID=new Map(members.map(m=>[m.id,m]));
 const donations=data.payments.filter(p=>p.kind==="donation"&&!p.void&&byID.has(p.member_id)).sort((a,b)=>b.paid_on.localeCompare(a.paid_on)||a.id.localeCompare(b.id));
 return <section aria-label="Donations">
  <div className="membership-summary"><article><span>Total donations · all time</span><strong>{money(donations.reduce((sum,p)=>sum+p.amount,0))}</strong></article></div>
  <p className="membership-note">Voluntary contributions are separate from monthly dues and card payments.</p>
  {donations.length?<ScrollTable className="membership-table-scroll"><table className="membership-table"><caption className="visually-hidden">Donations · all time</caption><thead><tr><th scope="col">Player</th><th scope="col">Amount</th><th scope="col">Date</th><th scope="col">Purpose / note</th>{(onEdit||onUndo)&&<th scope="col">Actions</th>}</tr></thead><tbody>{donations.map(p=>{const member=byID.get(p.member_id)!;return <tr key={p.id} className={member.active?undefined:"membership-row-inactive"}><th scope="row" className="member-name-cell">{onSelect?<button className="text-link" onClick={()=>onSelect(member)}>{member.name}</button>:member.name}{!member.active&&<small className="membership-inactive">Inactive</small>}</th><td>{money(p.amount)}</td><td>{p.paid_on||"Not recorded"}</td><td>{p.note||"—"}</td>{(onEdit||onUndo)&&<td><div className="membership-buttons">{onEdit&&<Button size="small" onClick={()=>onEdit(p,member)}>Edit donation</Button>}{onUndo&&<Button size="small" color="warning" onClick={()=>onUndo(p,member)}>Undo payment</Button>}</div></td>}</tr>;})}</tbody></table></ScrollTable>:<p className="membership-note">No donations recorded{members.length===1?" for this player":" for the selected players"}.</p>}
 </section>;
}
