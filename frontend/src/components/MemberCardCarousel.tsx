import { Children, useEffect, useId, useRef, useState, type ReactNode } from "react";

export default function MemberCardCarousel({children}:{children:ReactNode}) {
 const slides=Children.toArray(children),[index,setIndex]=useState(0),ref=useRef<HTMLDivElement>(null),id=useId();
 const previewTimers=useRef<ReturnType<typeof setTimeout>[]>([]);
 const stopPreview=()=>{previewTimers.current.forEach(clearTimeout);previewTimers.current=[];};
 useEffect(()=>{
  if(slides.length<2||matchMedia("(prefers-reduced-motion: reduce)").matches)return;
  previewTimers.current=[setTimeout(()=>{const el=ref.current;if(el&&document.visibilityState==="visible")el.scrollTo({left:el.clientWidth,behavior:"smooth"});},1000),setTimeout(()=>{ref.current?.scrollTo({left:0,behavior:"smooth"});},2300)];
  return stopPreview;
 },[slides.length]);
 const go=(next:number)=>{stopPreview();const el=ref.current;if(el)el.scrollTo({left:el.clientWidth*next,behavior:matchMedia("(prefers-reduced-motion: reduce)").matches?"instant":"smooth"});};
 return <section onPointerDownCapture={stopPreview} onKeyDownCapture={stopPreview} onWheelCapture={stopPreview} className="member-card-carousel" aria-label="Member summary cards" aria-roledescription="carousel">
  <div className="member-card-carousel-controls"><button type="button" title="Previous card" aria-label="Previous summary card" aria-controls={id} disabled={index===0} onClick={()=>go(index-1)}>‹</button><span aria-live="polite">{index===0?"Cards":"Outstanding dues"} · {index+1} of {slides.length}</span><button type="button" title="Next card" aria-label="Next summary card" aria-controls={id} disabled={index===slides.length-1} onClick={()=>go(index+1)}>›</button></div>
  <div id={id} className="member-card-carousel-track" ref={ref} onScroll={e=>{const el=e.currentTarget;setIndex(Math.max(0,Math.min(slides.length-1,Math.round(el.scrollLeft/el.clientWidth))));}}>
   {slides.map((slide,i)=><div key={i} className="member-card-carousel-slide" role="group" aria-roledescription="slide" aria-label={`${i+1} of ${slides.length}`} inert={i!==index}>{slide}</div>)}
  </div>
  <p className="member-card-carousel-hint">Swipe or use the arrows · Tap the card to flip</p>
 </section>;
}
