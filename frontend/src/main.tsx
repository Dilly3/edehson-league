import React from "react";
import ReactDOM from "react-dom/client";
import App from "./App";
import { ViewerGate } from "./components/ViewerAccess";
import { LeagueThemeProvider } from "./theme";
import "./styles.css";

ReactDOM.createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <LeagueThemeProvider><ViewerGate><App /></ViewerGate></LeagueThemeProvider>
  </React.StrictMode>,
);
