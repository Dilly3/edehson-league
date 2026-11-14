import ScrollTable from "./ScrollTable";
import MemberCardPayments from "./MemberCardPayments";
import PeriodPicker from "./PeriodPicker";
import { useState } from "react";
import { Alert, Button, CircularProgress, MenuItem, TextField } from "@mui/material";
import { duesMonthLabel, duesStatusLabel, money, memberDuesForYear, duesYearTotals, watDate, type Card, type Payment, type Due, type Member, type RegisterData } from "../membership";

export default function MemberPayments({member,data,year,onYearChange,loading,error,onRetry,admin,onAction,onEditDuesPayment,onUndoPayment,onEditCardPayment,onEditCard,onViewReceipt}:{
 member:Member;data:RegisterData;year:number;onYearChange:(year:number)=>void;loading:boolean;error:string;onRetry:()=>void;admin:boolean;onAction:(type:string,due:Due)=>void;onEditDuesPayment?:(payment:Payment)=>void;onViewReceipt?:(payment:Payment)=>void;onUndoPayment?:(payment:Payment)=>void;onEditCardPayment?:(payment:Payment)=>void;onEditCard?:(card:Card,index:number)=>void;
}){
 const [paymentType,setPaymentType]=useState("monthly");
 const [filter,setFilter]=useState("all"),[month,setMonth]=useState("all");
 const dues=memberDuesForYear(data.dues,member.id,year);
 const periodDues=dues.filter(d=>month==="all"||d.month.endsWith(`-${month}`));
 const periodLabel=month==="all"?year:duesMonthLabel(`${year}-${month}`);
 const totals=duesYearTotals(periodDues);
 const rows=periodDues.filter(d=>filter==="all"||d.status===filter);
 const latestYear=Math.max(Number(watDate().slice(0,4)),year);
 return <section className="member-payments" aria-labelledby="member-payments-heading">
  <header className="member-payments-header"><div><span className="eyebrow">{paymentType==="monthly"?"Monthly dues":`${paymentType} cards`}</span><h3 id="member-payments-heading">Payments</h3></div>
   <div className="member-payments-filters"><TextField select label="Payment type" size="small" value={paymentType} onChange={e=>setPaymentType(e.target.value)}><MenuItem value="monthly">Monthly dues</MenuItem><MenuItem value="yellow">Yellow cards</MenuItem><MenuItem value="red">Red cards</MenuItem></TextField><PeriodPicker year={year} month={month} minYear={2026} maxYear={latestYear} onChange={(y,m)=>{setMonth(m);if(y!==year)onYearChange(y);}}/>
    {paymentType==="monthly"&&<TextField select label="Show" size="small" value={filter} onChange={e=>setFilter(e.target.value)}><MenuItem value="all">All dues</MenuItem><MenuItem value="paid">Paid dues</MenuItem><MenuItem value="unpaid">Outstanding dues</MenuItem><MenuItem value="waived">Waived dues</MenuItem></TextField>}
   </div>
  </header>
  {loading?<div className="loading" role="status"><CircularProgress size={22}/><span>Loading {year} payments…</span></div>:error?<Alert severity="error" action={<Button onClick={onRetry}>Retry</Button>}>{error}</Alert>:paymentType!=="monthly"?<MemberCardPayments showPlayer={false} onUndo={admin&&onUndoPayment?(payment)=>onUndoPayment(payment):undefined} data={data} members={[member]} color={paymentType as "yellow"|"red"} year={year} month={month} onEdit={admin&&onEditCardPayment?(payment)=>onEditCardPayment(payment):undefined} onEditCard={admin&&onEditCard?(card,index)=>onEditCard(card,index):undefined}/>:<>
   <dl className="member-payment-totals"><div><dt>Paid · {periodLabel}</dt><dd>{money(totals.paid)}</dd></div><div><dt>Outstanding · {periodLabel}</dt><dd>{money(totals.outstanding)}</dd></div><div><dt>Waived · {periodLabel}</dt><dd>{money(totals.waived)}</dd></div></dl>
   <ScrollTable className="membership-table-scroll" tabIndex={0} aria-label="Payments table, scroll horizontally for more columns"><table className={`membership-table member-payments-table${!admin?" member-payments-viewer":""}`}><caption className="visually-hidden">{member.name}: {year} {filter==="all"?"monthly dues":filter+" dues"}</caption>
    <thead><tr><th scope="col">Month</th><th scope="col">Amount</th><th scope="col">Status</th>{admin&&<th scope="col">Payment date</th>}{admin&&<th scope="col">Manage</th>}</tr></thead>
    <tbody>{rows.map(d=>{
     const payment=data.payments.find(p=>p.due_id===d.id&&!p.void);
     return <tr key={d.id}><th scope="row">{duesMonthLabel(d.month)}</th><td>{money(d.amount)}</td><td><span className={`member-payment-status payment-status-${d.status}`} aria-label={d.status} title={d.auto_waived?"Automatically waived: before first paid dues month":d.status}>{duesStatusLabel(d.status,admin)}</span></td>{admin&&<td>{payment?(payment.paid_on||"Not recorded"):"—"}</td>}{admin&&<td><div className="membership-buttons">{d.status==="unpaid"&&<><Button size="small" onClick={()=>onAction("pay_due",d)}>Mark paid</Button><Button size="small" onClick={()=>onAction("waive_due",d)}>Waive</Button></>}{d.status==="waived"&&!d.auto_waived&&<Button size="small" onClick={()=>onAction("reopen_due",d)}>Reopen</Button>}{d.status==="paid"&&payment?.receipt_request_id&&onViewReceipt&&<Button size="small" onClick={()=>onViewReceipt(payment)}>Receipt</Button>}{d.status==="paid"&&payment&&onEditDuesPayment&&<Button size="small" onClick={()=>onEditDuesPayment(payment)}>Edit payment date</Button>}{d.status==="paid"&&payment&&onUndoPayment&&<Button size="small" color="warning" onClick={()=>onUndoPayment(payment)}>Undo payment</Button>}</div></td>}</tr>;
    })}</tbody>
   </table></ScrollTable>
   {!rows.length&&<p className="member-payments-empty">{dues.length?"No dues match this filter.":`No monthly dues have been issued for ${year}.`}</p>}
   <p className="member-payments-legend"><span>✅ Paid</span><span>⌛️ Outstanding</span><span>Waived dues carry no balance</span></p>
  </>}
 </section>;
}
