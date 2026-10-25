import { useEffect, useId, useRef, useState, type HTMLAttributes } from "react";

export default function ScrollTable({children,className="",...props}:HTMLAttributes<HTMLDivElement>) {
 const id=useId(),ref=useRef<HTMLDivElement>(null);
 const [edges,setEdges]=useState({left:false,right:false});
 useEffect(()=>{
  const el=ref.current;if(!el)return;
  const update=()=>{const left=el.scrollLeft>2,right=el.scrollLeft+el.clientWidth<el.scrollWidth-2;setEdges(old=>old.left===left&&old.right===right?old:{left,right});};
  const resize=new ResizeObserver(update);resize.observe(el);if(el.firstElementChild)resize.observe(el.firstElementChild);
  const mutation=new MutationObserver(update);mutation.observe(el,{subtree:true,childList:true,characterData:true});
  el.addEventListener("scroll",update,{passive:true});update();
  return()=>{resize.disconnect();mutation.disconnect();el.removeEventListener("scroll",update);};
 },[]);
 const scroll=(direction:number)=>{const el=ref.current;if(el)el.scrollBy({left:direction*el.clientWidth*.7,behavior:matchMedia("(prefers-reduced-motion: reduce)").matches?"instant":"smooth"});};
 return <div className="table-scroll-shell">
  {(edges.left||edges.right)&&<div className="table-scroll-controls"><button type="button" aria-label="Scroll table left" aria-controls={id} disabled={!edges.left} onClick={()=>scroll(-1)}>‹</button><span>{edges.right?"More columns →":"← More columns"}</span><button type="button" aria-label="Scroll table right" aria-controls={id} disabled={!edges.right} onClick={()=>scroll(1)}>›</button></div>}
  <div {...props} id={id} ref={ref} className={`membership-table-scroll ${className}`} tabIndex={0}>{children}</div>
 </div>;
}
