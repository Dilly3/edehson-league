import React from "react";
import ReactDOM from "react-dom/client";
import App from "./App";
import UpdatePrompt from "./components/UpdatePrompt";
import { LeagueThemeProvider } from "./theme";
import "./styles.css";

ReactDOM.createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <LeagueThemeProvider><App /><UpdatePrompt /></LeagueThemeProvider>
  </React.StrictMode>,
);
