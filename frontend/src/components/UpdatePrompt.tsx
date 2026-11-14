import { useEffect, useState } from "react";
import { Button, Snackbar } from "@mui/material";

const CHECK_EVERY_MS = 5 * 60 * 1000;

async function deployedVersion(): Promise<string | null> {
  try {
    const response = await fetch(`/version.json?t=${Date.now()}`, { cache: "no-store" });
    if (!response.ok) return null; // e.g. the dev server, which has no version file
    const body = await response.json() as { version?: unknown };
    return typeof body.version === "string" ? body.version : null;
  } catch { return null; }
}

/** Shows "A new version is available" when a newer build has been deployed while this tab was open. */
export default function UpdatePrompt() {
  const [available, setAvailable] = useState(false);
  useEffect(() => {
    if (import.meta.env.DEV) return;
    let stopped = false;
    const check = async () => {
      if (stopped || document.visibilityState !== "visible") return;
      const latest = await deployedVersion();
      if (latest && latest !== __APP_VERSION__) setAvailable(true);
    };
    const timer = window.setInterval(() => void check(), CHECK_EVERY_MS);
    const onVisible = () => void check();
    document.addEventListener("visibilitychange", onVisible);
    window.addEventListener("focus", onVisible);
    // A page left open across a deploy can fail to load a code chunk that no longer exists.
    const onChunkError = (event: Event) => { event.preventDefault(); window.location.reload(); };
    window.addEventListener("vite:preloadError", onChunkError);
    void check();
    return () => {
      stopped = true;
      window.clearInterval(timer);
      document.removeEventListener("visibilitychange", onVisible);
      window.removeEventListener("focus", onVisible);
      window.removeEventListener("vite:preloadError", onChunkError);
    };
  }, []);
  return <Snackbar open={available} anchorOrigin={{ vertical: "top", horizontal: "center" }} className="update-prompt"
    message="A new version of GreatMindsSport is available."
    action={<Button color="secondary" size="small" variant="contained" onClick={() => window.location.reload()}>Refresh</Button>}/>;
}
