import type { PlayerStats, Standing } from "../types";

export default function LeagueLeaders({standings,stats,onTeamClick}:{standings:Standing[]|null;stats:PlayerStats[]|null;onTeamClick:(tag:string,name:string)=>void}) {
 const teams=(standings??[]).filter(t=>t.played>0);
 const categories=[
  {title:"Best scoring team",kind:"scoring",available:standings!==null,max:true,value:(t:Standing)=>t.goals_for,unit:"goals scored"},
  {title:"Best defending team",kind:"defending",available:standings!==null,max:false,value:(t:Standing)=>t.goals_against,unit:"goals conceded"},
  {title:"Fair play team",kind:"fair-play",available:standings!==null&&stats!==null,max:false,value:(t:Standing)=>(stats??[]).filter(p=>p.team_tag===t.team_tag).reduce((n,p)=>n+p.yellow_cards+p.red_cards,0),unit:"cards"},
 ];
 return <section className="league-leaders" aria-label="Team season leaders">
  {categories.map(c=>{
   const best=teams.length?(c.max?Math.max:Math.min)(...teams.map(c.value)):null;
   const leaders=best===null?[]:teams.filter(t=>c.value(t)===best).sort((a,b)=>a.team_name.localeCompare(b.team_name));
   return <article className={`league-leader league-leader-${c.kind}`} key={c.kind}><h3>{c.title}</h3>{!c.available?<p>Statistics unavailable</p>:leaders.length?<><div className="league-leader-teams">{leaders.map(t=><button className="text-link" key={t.team_tag} onClick={()=>onTeamClick(t.team_tag,t.team_name)}>{t.team_name}</button>)}</div><strong>{best} {c.unit}</strong>{leaders.length>1&&<small>Joint leaders</small>}</>:<p>Available after the first full-time result.</p>}</article>;
  })}
  <p className="league-leaders-note">Full-time matches only · Defence uses fewest goals conceded · Fair play uses fewest yellow + red cards, counting each card equally · Totals are not adjusted for matches played.</p>
 </section>;
}
