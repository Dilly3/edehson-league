export const VIEWER_IDLE_MS=2*60*60*1000;
// Polling never calls activity(). Only trusted UI events renew this timer.
export function createViewerIdleTimer({remaining=VIEWER_IDLE_MS,onIdle,onActivity}:{remaining?:number;onIdle:()=>void;onActivity:()=>void}) {
 let deadline=Date.now()+remaining,lastSent=0,stopped=false;
 let pending:ReturnType<typeof setTimeout>|undefined;
 const stop=()=>{stopped=true;clearInterval(timer);if(pending)clearTimeout(pending);};
 const check=()=>{if(!stopped&&Date.now()>=deadline){stop();onIdle();return false;}return !stopped;};
 const send=()=>{pending=undefined;if(!check())return;lastSent=Date.now();onActivity();};
 const activity=(notify=true)=>{
  if(!check())return false;
  deadline=Date.now()+VIEWER_IDLE_MS;
  if(!notify)return true;
  if(Date.now()-lastSent>=60000){if(pending)clearTimeout(pending);send();}
  else if(!pending)pending=setTimeout(send,60000-(Date.now()-lastSent));
  return true;
 };
 const timer=setInterval(check,1000);
 return {activity,check,stop};
}
