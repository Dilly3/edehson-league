import { describe,it,expect } from "vitest";
import { individualCardPayments,cardSummary,cardQuote,emptyRegister,nairaToKobo,nextMonthlyFeeDate,duesMonthLabel,duesStatusLabel,duesPaymentDate,memberDuesForYear,duesYearTotals,type RegisterData } from "./membership";
function register():RegisterData{return {...emptyRegister,cards:[{id:"old",member_id:"m",color:"yellow",count:10,unit_amount:50000,occurred_on:"2026-01-01",note:"",void:false},{id:"new",member_id:"m",color:"yellow",count:2,unit_amount:75000,occurred_on:"2026-02-01",note:"",void:false}],payments:[{id:"p",member_id:"m",kind:"yellow",due_id:null,amount:250000,paid_on:"2026-01-05",note:"",void:false}],allocations:[{id:"a",card_id:"old",payment_id:"p",count:5}]};}
describe("Great Minds Sport register calculations",()=>{
 it("preserves total cards while settlements reduce outstanding",()=>{const data=register();expect(cardSummary(data,"m","yellow")).toEqual({received:12,paid:5,outstanding:7,balance:400000});});
 it("settles oldest cards at historical rates",()=>{expect(cardQuote(register(),"m","yellow",6)).toBe(325000);expect(cardQuote(register(),"m","yellow",8)).toBeNull();});
 it("reversed payments restore outstanding cards",()=>{const data=register();data.payments[0].void=true;expect(cardSummary(data,"m","yellow")).toEqual({received:12,paid:0,outstanding:12,balance:650000});});
 it("voided card records are excluded from active totals",()=>{const data=register();data.cards[1].void=true;expect(cardSummary(data,"m","yellow").received).toBe(10);expect(cardSummary(data,"m","red").received).toBe(0);});
 it("parses fees in exact minor units",()=>{expect(nairaToKobo("1000")).toBe(100000);expect(nairaToKobo("500.25")).toBe(50025);expect(nairaToKobo("0.01")).toBe(1);for(const value of ["0","-1","1.001","1e3","NaN","100000001"])expect(()=>nairaToKobo(value)).toThrow();});
});

it("monthly fee changes begin in the next month, including year rollover",()=>{
 expect(nextMonthlyFeeDate("2026-10-23")).toBe("2026-11-01");
 expect(nextMonthlyFeeDate("2026-12-31")).toBe("2027-01-01");
});

it("formats dues periods and defaults payments to their first day",()=>{
 expect(duesMonthLabel("2026-01")).toBe("2026-JAN");
 expect(duesMonthLabel("2026-10")).toBe("2026-OCT");
 expect(duesPaymentDate("2026-01")).toBe("2026-01-01");
 expect(duesPaymentDate("2026-10")).toBe("2026-10-01");
});
it("distinguishes admin statuses from public payment icons",()=>{
 expect(duesStatusLabel("paid",true)).toBe("PAID");
 expect(duesStatusLabel("unpaid",true)).toBe("UNPAID");
 expect(duesStatusLabel("paid",false)).toBe("✅");
 expect(duesStatusLabel("unpaid",false)).toBe("⌛️");
 expect(duesStatusLabel("waived",false)).toBe("WAIVED");
});

it("selects only the member's chosen year and sorts dues by month",()=>{
 const dues=[{id:"b",member_id:"m",month:"2026-02",amount:100000,status:"unpaid",revision:0},{id:"a",member_id:"m",month:"2026-01",amount:100000,status:"paid",revision:0},{id:"old",member_id:"m",month:"2027-01",amount:150000,status:"paid",revision:0},{id:"other",member_id:"n",month:"2026-01",amount:100000,status:"paid",revision:0}] as RegisterData["dues"];
 expect(memberDuesForYear(dues,"m",2026).map(d=>d.id)).toEqual(["a","b"]);
});
it("keeps paid, outstanding and waived yearly totals separate",()=>{
 const dues=[{status:"paid",amount:100000},{status:"unpaid",amount:150000},{status:"waived",amount:100000}] as RegisterData["dues"];
 expect(duesYearTotals(dues)).toEqual({paid:100000,outstanding:150000,waived:100000});
 expect(duesYearTotals([])).toEqual({paid:0,outstanding:0,waived:0});
});

it("renders one row per card with stable active receipt allocations",()=>{
 const data=register();data.cards[0].count=6;data.allocations[0].count=2;
 data.payments.push({...data.payments[0],id:"reversed",void:true});
 data.allocations.push({id:"ignored",card_id:"old",payment_id:"reversed",count:3});
 const rows=individualCardPayments(data,data.cards[0]);
 expect(rows).toHaveLength(6);
 expect(rows.map(r=>r.index)).toEqual([0,1,2,3,4,5]);
 expect(rows.filter(r=>r.payment).length).toBe(2);
 expect(rows[0].payment?.id).toBe("p");
 expect(rows[2].payment).toBeUndefined();
});

it("allows advance payment without prefilling a future receipt date",()=>{
 const future=`${new Date().getFullYear()+1}-12`;
 expect(duesPaymentDate(future)).not.toBe(`${future}-01`);
});
