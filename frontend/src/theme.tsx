import { createContext, useContext, useLayoutEffect, useMemo, useState, type ReactNode } from "react";
import { CssBaseline, ThemeProvider, createTheme } from "@mui/material";

type Mode = "light" | "dark";
const AppearanceContext = createContext<{ mode: Mode; toggle: () => void }>({ mode: "light", toggle: () => {} });
export const useAppearance = () => useContext(AppearanceContext);

function initialMode(): Mode {
  try {
    const saved = localStorage.getItem("edehsons-color-mode");
    if (saved === "light" || saved === "dark") return saved;
  } catch { /* Storage may be blocked; the toggle still works for this visit. */ }
  return "light";
}

export function LeagueThemeProvider({ children }: { children: ReactNode }) {
  const [mode, setMode] = useState<Mode>(initialMode);
  const theme = useMemo(() => buildTheme(mode === "dark"), [mode]);
  useLayoutEffect(() => {
    document.documentElement.dataset.theme = mode;
    document.documentElement.style.colorScheme = mode;
    const meta = document.querySelector('meta[name="theme-color"]');
    meta?.setAttribute("content", mode === "dark" ? "#111814" : "#0d3b2a");
  }, [mode]);
  const toggle = () => {
    const next = mode === "dark" ? "light" : "dark";
    setMode(next);
    try { localStorage.setItem("edehsons-color-mode", next); } catch { /* Optional persistence. */ }
  };
  return <AppearanceContext.Provider value={{ mode, toggle }}>
    <ThemeProvider theme={theme}><CssBaseline />{children}</ThemeProvider>
  </AppearanceContext.Provider>;
}

const body =
  '"Barlow", "Helvetica Neue", Helvetica, Arial, system-ui, sans-serif';
const display =
  '"Barlow Condensed", "Arial Narrow", "Helvetica Neue", Arial, sans-serif';

function buildTheme(dark: boolean) { return createTheme({
  palette: {
    mode: dark ? "dark" : "light",
    primary: { main: dark ? "#b7dc78" : "#0d3b2a", contrastText: dark ? "#122016" : "#ffffff" },
    secondary: { main: "#d4f23a", contrastText: "#0d3b2a" },
    error: { main: dark ? "#ff9d8a" : "#b8321f" },
    background: { default: dark ? "#111814" : "#f3f0e8", paper: dark ? "#1b251e" : "#fffdf8" },
    text: { primary: dark ? "#f2f1e9" : "#15171a", secondary: dark ? "#b9c4bc" : "#4a4f55" },
    divider: dark ? "#3c4b40" : "#d9d4c7",
  },
  shape: { borderRadius: 4 },
  typography: {
    fontFamily: body,
    fontSize: 15,
    button: {
      fontFamily: display,
      textTransform: "uppercase",
      fontWeight: 700,
      letterSpacing: "0.06em",
      fontSize: "0.95rem",
    },
  },
  components: {
    MuiButton: {
      defaultProps: { disableElevation: true },
      styleOverrides: {
        root: { borderRadius: 3, padding: "8px 16px", lineHeight: 1.2 },
        outlined: { borderColor: dark ? "#a9b5ac" : "#15171a", color: dark ? "#f2f1e9" : "#15171a" },
        text: { color: dark ? "#b7dc78" : "#0d3b2a" },
      },
    },
    MuiTextField: { defaultProps: { size: "small", fullWidth: true } },
    MuiOutlinedInput: {
      styleOverrides: {
        root: { background: dark ? "#162019" : "#fffdf8" },
        notchedOutline: { borderColor: dark ? "#697a6d" : "#b9b3a3" },
      },
    },
    MuiDialog: {
      defaultProps: { fullWidth: true, maxWidth: "md" },
      styleOverrides: {
        paper: {
          borderRadius: 6,
          border: dark ? "1px solid #697a6d" : "1px solid #15171a",
          backgroundImage: "none",
        },
      },
    },
    MuiDialogTitle: {
      styleOverrides: {
        root: {
          fontFamily: display,
          fontWeight: 800,
          fontSize: "1.9rem",
          lineHeight: 1.1,
          textTransform: "uppercase",
          padding: "24px 24px 8px",
        },
      },
    },
    MuiDialogActions: { styleOverrides: { root: { padding: "12px 24px 20px" } } },
    MuiTab: {
      styleOverrides: {
        root: {
          fontFamily: display,
          fontWeight: 700,
          fontSize: "1rem",
          letterSpacing: "0.06em",
          textTransform: "uppercase",
        },
      },
    },
    MuiTabs: {
      styleOverrides: { indicator: { height: 3, backgroundColor: dark ? "#b7dc78" : "#0d3b2a" } },
    },
    MuiAlert: {
      styleOverrides: { root: { borderRadius: 3, fontSize: "0.9rem" } },
    },
    MuiSnackbar: { defaultProps: { anchorOrigin: { vertical: "bottom", horizontal: "left" } } },
    MuiTableCell: {
      styleOverrides: {
        root: {
          borderBottom: dark ? "1px solid #344238" : "1px solid #e4dfd2",
          padding: "14px 12px",
          fontSize: "0.95rem",
        },
        head: {
          background: "transparent",
          color: dark ? "#b9c4bc" : "#4a4f55",
          fontFamily: display,
          fontSize: "0.9rem",
          fontWeight: 700,
          textTransform: "uppercase",
          letterSpacing: ".08em",
          borderBottom: dark ? "2px solid #b9c4bc" : "2px solid #15171a",
          whiteSpace: "nowrap",
        },
      },
    },
  },
});
}
