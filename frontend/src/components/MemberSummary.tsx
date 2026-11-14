import { cardSummary, duesMonthLabel, money, watDate, type RegisterData } from "../membership";

/** At-a-glance balances for one player: outstanding dues plus yellow and red card fines.
 * Replaces the tap-to-flip carousel so nothing is hidden behind a gesture. */
export default function MemberSummary({ data, memberID, year, duesLoading, duesError }: {
  data: RegisterData; memberID: string; year: number; duesLoading: boolean; duesError: string;
}) {
  const unpaid = data.dues
    .filter(d => d.member_id === memberID && d.month.startsWith(`${year}-`) && d.month <= watDate().slice(0, 7) && d.status === "unpaid")
    .sort((a, b) => a.month.localeCompare(b.month));
  const duesOwed = unpaid.reduce((sum, d) => sum + d.amount, 0);
  const months = unpaid.map(d => duesMonthLabel(d.month).split("-")[1]?.trim() ?? d.month);
  const cards = (["yellow", "red"] as const).map(color => ({ color, ...cardSummary(data, memberID, color) }));
  const totalOwed = duesOwed + cards.reduce((sum, c) => sum + c.balance, 0);

  return <section className="member-summary" aria-label="Balance summary">
    <div className={`member-summary-total${totalOwed ? " owes" : ""}`}>
      <span>Total outstanding</span>
      <strong>{duesLoading ? "…" : money(totalOwed)}</strong>
      <small>{totalOwed ? "Dues through this month plus unpaid card fines" : "All settled"}</small>
    </div>
    <div className="member-summary-tiles">
      <article className="member-summary-tile dues">
        <header><i aria-hidden="true" /> Monthly dues · {year}</header>
        {duesLoading ? <p role="status">Loading dues…</p>
          : duesError ? <p role="alert">Could not load dues.</p>
          : <>
            <strong>{money(duesOwed)}</strong>
            <p>{months.length ? <>Unpaid: {months.join(", ")}</> : "No outstanding dues"}</p>
          </>}
      </article>
      {cards.map(c => <article key={c.color} className={`member-summary-tile ${c.color}`}>
        <header><i aria-hidden="true" /> {c.color === "yellow" ? "Yellow" : "Red"} cards</header>
        <strong>{money(c.balance)}</strong>
        <p>{c.received
          ? <><b>{c.received}</b> received · <b>{c.paid}</b> paid · <b>{c.outstanding}</b> unpaid</>
          : "None received"}</p>
      </article>)}
    </div>
  </section>;
}
