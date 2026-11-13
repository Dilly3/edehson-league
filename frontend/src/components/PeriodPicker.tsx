import { useState } from "react";
import { Button, IconButton, InputAdornment, Popover, TextField } from "@mui/material";
import ChevronLeftRounded from "@mui/icons-material/ChevronLeftRounded";
import ChevronRightRounded from "@mui/icons-material/ChevronRightRounded";
import CalendarMonthOutlined from "@mui/icons-material/CalendarMonthOutlined";

const MONTHS = ["Jan","Feb","Mar","Apr","May","Jun","Jul","Aug","Sep","Oct","Nov","Dec"];

/** One field for year + month. `month` is "all" or a two-digit month ("01"–"12"). */
export default function PeriodPicker({ year, month, minYear, maxYear, onChange, label = "Period" }: {
  year: number; month: string; minYear: number; maxYear: number;
  onChange: (year: number, month: string) => void; label?: string;
}) {
  const [anchor, setAnchor] = useState<HTMLElement | null>(null);
  const [viewYear, setViewYear] = useState(year);
  const open = (el: HTMLElement) => { setViewYear(year); setAnchor(el); };
  const pick = (m: string) => { onChange(viewYear, m); setAnchor(null); };
  const display = month === "all" ? `${year} · All months` : `${MONTHS[Number(month) - 1]} ${year}`;

  return <>
    <TextField className="period-picker-field" label={label} value={display} size="small"
      onClick={e => open(e.currentTarget)}
      onKeyDown={e => { if (e.key === "Enter" || e.key === " " || e.key === "ArrowDown") { e.preventDefault(); open(e.currentTarget); } }}
      slotProps={{
        input: { readOnly: true, endAdornment: <InputAdornment position="end"><CalendarMonthOutlined fontSize="small" /></InputAdornment> },
        htmlInput: { "aria-haspopup": "dialog", style: { cursor: "pointer" } },
      }} />
    <Popover open={!!anchor} anchorEl={anchor} onClose={() => setAnchor(null)}
      anchorOrigin={{ vertical: "bottom", horizontal: "left" }} transformOrigin={{ vertical: "top", horizontal: "left" }}
      slotProps={{ paper: { className: "period-picker", role: "dialog", "aria-label": "Choose period" } as object }}>
      <div className="period-picker-head">
        <IconButton size="small" aria-label="Previous year" disabled={viewYear <= minYear} onClick={() => setViewYear(y => y - 1)}><ChevronLeftRounded /></IconButton>
        <strong aria-live="polite">{viewYear}</strong>
        <IconButton size="small" aria-label="Next year" disabled={viewYear >= maxYear} onClick={() => setViewYear(y => y + 1)}><ChevronRightRounded /></IconButton>
      </div>
      <Button fullWidth size="small" className={viewYear === year && month === "all" ? "period-all selected" : "period-all"}
        aria-pressed={viewYear === year && month === "all"} onClick={() => pick("all")}>All months</Button>
      <div className="period-picker-grid">
        {MONTHS.map((name, i) => {
          const value = String(i + 1).padStart(2, "0");
          const selected = viewYear === year && month === value;
          return <button key={name} type="button" className={selected ? "selected" : undefined}
            aria-pressed={selected} aria-label={`${name} ${viewYear}`} onClick={() => pick(value)}>{name}</button>;
        })}
      </div>
    </Popover>
  </>;
}
