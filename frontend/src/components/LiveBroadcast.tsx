import { useEffect, useRef, useState } from "react";
import { Alert, Button } from "@mui/material";

type Player = {
  attachHTMLVideoElement: (video: HTMLVideoElement) => void;
  load: (url: string) => void;
  play: () => void;
  delete: () => void;
  addEventListener: (event: string, callback: () => void) => void;
};
type SDK = { isPlayerSupported: boolean; create: () => Player; PlayerEventType: { ERROR: string } };
let sdkPromise: Promise<SDK> | undefined;
function loadSDK() {
  if (!sdkPromise) sdkPromise = new Promise<SDK>((resolve, reject) => {
    const script = document.createElement("script");
    script.src = "https://player.live-video.net/1.57.0/amazon-ivs-player.min.js";
    script.async = true;
    script.onload = () => {
      const sdk = (window as unknown as { IVSPlayer?: SDK }).IVSPlayer;
      if (sdk) resolve(sdk); else reject(new Error("Player unavailable"));
    };
    script.onerror = () => { script.remove(); reject(new Error("Player download failed")); };
    document.head.append(script);
  }).catch(error => { sdkPromise = undefined; throw error; });
  return sdkPromise;
}
export default function LiveBroadcast({ url }: { url: string }) {
  const video = useRef<HTMLVideoElement>(null);
  const [started, setStarted] = useState(false);
  const [attempt, setAttempt] = useState(0);
  const [error, setError] = useState("");
  useEffect(() => {
    if (!started) return;
    let active = true;
    let player: Player | undefined;
    setError("");
    void loadSDK().then(sdk => {
      if (!active || !video.current) return;
      if (!sdk.isPlayerSupported) { setError("This browser cannot play this broadcast. Try an updated Safari or Chrome browser."); return; }
      player = sdk.create();
      player.attachHTMLVideoElement(video.current);
      player.addEventListener(sdk.PlayerEventType.ERROR, () => {
        if (active) setError("The broadcast is offline or the connection was interrupted. Try again when the broadcaster is streaming.");
      });
      player.load(url);
      player.play();
    }).catch(() => { if (active) setError("Could not load the video player. Check your connection and try again."); });
    return () => { active = false; player?.delete(); };
  }, [url, started, attempt]);
  return <section className="live-broadcast" aria-label="Match broadcast">
    <h3>Watch live</h3>
    <p className="muted">Video is available while the broadcaster is streaming. Match scores are updated separately.</p>
    {!started ? <Button variant="contained" onClick={() => setStarted(true)}>Watch broadcast</Button> : <>
      <video ref={video} controls playsInline muted aria-label="Live match video" style={{width:"100%", aspectRatio:"16 / 9", background:"#000", borderRadius:12}} />
      {error && <Alert severity="info" action={<Button onClick={() => setAttempt(n => n + 1)}>Retry</Button>}>{error}</Alert>}
      <p className="muted">Playback starts muted. Use the player controls to turn on sound.</p>
    </>}
  </section>;
}
