export interface Member { id:string; name:string; joined_on:string; active:boolean; revision:number }
export interface Fee { id:string; kind:"monthly"|"yellow"|"red"; effective_from:string; amount:number }
export interface Due { auto_waived?:boolean; id:string; member_id:string; month:string; amount:number; status:"unpaid"|"paid"|"waived"; revision:number }
export interface Card { id:string; member_id:string; color:"yellow"|"red"; count:number; unit_amount:number; occurred_on:string; note:string; void:boolean }
export interface Payment { id:string; member_id:string; kind:string; due_id:string|null; amount:number; paid_on:string; note:string; void:boolean }
export interface Allocation { id:string; payment_id:string; card_id:string; count:number }
export interface AnnualFee {id:string; year:number; amount:number; revision:number; locked:boolean}
export interface RegisterData { annual_fees?:AnnualFee[]; members:Member[]; fees:Fee[]; dues:Due[]; cards:Card[]; payments:Payment[]; allocations:Allocation[] }
export interface History { id:string; type:string; note:string; before:string; after:string; created_at:string }
export const emptyRegister:RegisterData={members:[],fees:[],dues:[],cards:[],payments:[],allocations:[]};
export const money=(amount:number)=>new Intl.NumberFormat("en-NG",{style:"currency",currency:"NGN",maximumFractionDigits:2}).format(amount/100);
export const watDate=()=>new Intl.DateTimeFormat("en-CA",{timeZone:"Africa/Lagos",year:"numeric",month:"2-digit",day:"2-digit"}).format(new Date());
export function nairaToKobo(value:string) {
 if(!/^\d+(\.\d{1,2})?$/.test(value.trim())) throw new Error("Enter a positive amount with at most two decimal places.");
 const [whole,fraction=""]=value.trim().split(".");
 const amount=Number(whole)*100+Number(fraction.padEnd(2,"0"));
 if(!Number.isSafeInteger(amount)||amount<1||amount>10000000000)throw new Error("Fee must be positive and no more than ₦100,000,000.");
 return amount;
}
export function cardSummary(data:RegisterData, memberID:string,color:"yellow"|"red"){
 const payments=new Set(data.payments.filter(p=>!p.void).map(p=>p.id));
 const cards=data.cards.filter(c=>c.member_id===memberID&&c.color===color&&!c.void);
 return cards.reduce((s,c)=>{
  const paid=data.allocations.filter(a=>a.card_id===c.id&&payments.has(a.payment_id)).reduce((n,a)=>n+a.count,0);
  return {received:s.received+c.count,paid:s.paid+paid,outstanding:s.outstanding+c.count-paid,balance:s.balance+(c.count-paid)*c.unit_amount};
 },{received:0,paid:0,outstanding:0,balance:0});
}
export function cardQuote(data:RegisterData,memberID:string,color:"yellow"|"red",count:number){
 let remaining=count,total=0;
 const paidIDs=new Set(data.payments.filter(p=>!p.void).map(p=>p.id));
 for(const c of data.cards.filter(c=>c.member_id===memberID&&c.color===color&&!c.void).sort((a,b)=>a.occurred_on.localeCompare(b.occurred_on)||a.id.localeCompare(b.id))){
  const paid=data.allocations.filter(a=>a.card_id===c.id&&paidIDs.has(a.payment_id)).reduce((n,a)=>n+a.count,0);
  const settled=Math.min(c.count-paid,remaining);total+=settled*c.unit_amount;remaining-=settled;if(!remaining)break;
 }
 return remaining===0?total:null;
}

export function nextMonthlyFeeDate(date=watDate()) {
 const d=new Date(`${date.slice(0,7)}-01T00:00:00Z`);
 d.setUTCMonth(d.getUTCMonth()+1);
 return d.toISOString().slice(0,10);
}

export function duesMonthLabel(month:string) {
 const [year,value]=month.split("-");
 const names=["JAN","FEB","MAR","APR","MAY","JUN","JUL","AUG","SEP","OCT","NOV","DEC"];
 return names[Number(value)-1]?`${year}-${names[Number(value)-1]}`:month;
}
export function duesStatusLabel(status:Due["status"],admin:boolean) {
 return admin?status.toUpperCase():status==="paid"?"✅":status==="unpaid"?"⌛️":"WAIVED";
}
export function duesPaymentDate(month:string) { const first=`${month}-01`; return first>watDate()?watDate():first; }
export function duesPaymentMinimum(month:string,joined:string) {return [month+"-01",joined.slice(0,7)+"-01"].sort()[0];}

export function memberDuesForYear(dues:Due[],memberID:string,year:number) {
 return dues.filter(d=>d.member_id===memberID&&d.month.startsWith(`${year}-`)).sort((a,b)=>a.month.localeCompare(b.month));
}
export function duesYearTotals(dues:Due[]) {
 return dues.reduce((totals,d)=>{
  if(d.status==="paid")totals.paid+=d.amount;
  if(d.status==="unpaid")totals.outstanding+=d.amount;
  if(d.status==="waived")totals.waived+=d.amount;
  return totals;
 },{paid:0,outstanding:0,waived:0});
}

// Grouped historical cards have no individual identities in the source. Use a
// stable allocation order until an admin splits the group by editing a card.
export function individualCardPayments(data:RegisterData,card:Card) {
 const payments=new Map(data.payments.filter(p=>!p.void).map(p=>[p.id,p]));
 const assigned=data.allocations.filter(a=>a.card_id===card.id&&payments.has(a.payment_id)).sort((a,b)=>a.id.localeCompare(b.id)).flatMap(a=>Array.from({length:a.count},()=>payments.get(a.payment_id)!));
 return Array.from({length:card.count},(_,index)=>({index,payment:assigned[index]}));
}
