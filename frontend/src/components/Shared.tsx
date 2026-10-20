import { teamInitials } from "../football";
import type { ReactNode } from "react";
import {
  Button,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
} from "@mui/material";
import ArrowForwardRounded from "@mui/icons-material/ArrowForwardRounded";
import type { Standing } from "../types";
export function Crest({
  tag,
  name,
  size = 36,
}: {
  tag: string;
  name?: string;
  size?: number;
}) {
  const n = Array.from(tag).reduce((s, c) => s + c.charCodeAt(0), 0) % 6;
  return (
    <span
      aria-hidden="true"
      className={`crest crest-${n}`}
      style={{ width: size, height: size, fontSize: size * 0.4 }}
    >
      {teamInitials(name || tag)}
    </span>
  );
}
export function Empty({
  title,
  detail,
  action,
}: {
  title: string;
  detail: string;
  action?: ReactNode;
}) {
  return (
    <div className="empty">
      <h3>{title}</h3>
      <p>{detail}</p>
      {action}
    </div>
  );
}
export function SectionHeading({
  eyebrow,
  title,
  action,
}: {
  eyebrow?: string;
  title: string;
  action?: ReactNode;
}) {
  return (
    <div className="section-heading">
      <div>
        {eyebrow && <span className="eyebrow">{eyebrow}</span>}
        <h2>{title}</h2>
      </div>
      {action}
    </div>
  );
}
export function MoreButton({
  children,
  onClick,
}: {
  children: ReactNode;
  onClick: () => void;
}) {
  return (
    <Button
      size="small"
      endIcon={<ArrowForwardRounded fontSize="small" />}
      onClick={onClick}
    >
      {children}
    </Button>
  );
}
export function LeagueTable({
  rows,
  compact = false,
  onTeamClick,
}: {
  rows: Standing[];
  compact?: boolean;
  onTeamClick?: (tag: string, name: string) => void;
}) {
  if (!rows.length)
    return (
      <Empty
        title="The table starts here"
        detail="Teams will appear once they have been registered."
      />
    );
  return (
    <TableContainer>
      <Table size="small" aria-label="League standings" className="league-table">
        <TableHead>
          <TableRow>
            <TableCell className="col-pos">#</TableCell>
            <TableCell>Club</TableCell>
            <TableCell align="center">P</TableCell>
            {!compact && (
              <>
                <TableCell align="center">W</TableCell>
                <TableCell align="center">D</TableCell>
                <TableCell align="center">L</TableCell>
                <TableCell align="center">GF</TableCell>
                <TableCell align="center">GA</TableCell>
              </>
            )}
            <TableCell align="center">GD</TableCell>
            <TableCell align="center">Pts</TableCell>
          </TableRow>
        </TableHead>
        <TableBody>
          {rows.map((row, i) => (
            <TableRow
              key={row.team_tag}
              className={i === 0 ? "table-leader" : ""}
            >
              <TableCell className="col-pos">
                <span className={i === 0 ? "rank rank-first" : "rank"}>
                  {row.position}
                </span>
              </TableCell>
              <TableCell>
                <div className="club-cell">
                  <Crest tag={row.team_tag} name={row.team_name} size={30} />
                  <div>
                    {onTeamClick ? (
                      <button
                        className="text-link"
                        onClick={() => onTeamClick(row.team_tag, row.team_name)}
                        aria-label={`View ${row.team_name} roster`}
                      >
                        {row.team_name}
                      </button>
                    ) : (
                      <strong>{row.team_name}</strong>
                    )}
                    {!compact && <small>{row.team_tag}</small>}
                  </div>
                </div>
              </TableCell>
              <TableCell align="center" className="num">
                {row.played}
              </TableCell>
              {!compact && (
                <>
                  <TableCell align="center" className="num">
                    {row.won}
                  </TableCell>
                  <TableCell align="center" className="num">
                    {row.drawn}
                  </TableCell>
                  <TableCell align="center" className="num">
                    {row.lost}
                  </TableCell>
                  <TableCell align="center" className="num">
                    {row.goals_for}
                  </TableCell>
                  <TableCell align="center" className="num">
                    {row.goals_against}
                  </TableCell>
                </>
              )}
              <TableCell align="center" className="num">
                {row.goal_difference > 0 ? "+" : ""}
                {row.goal_difference}
              </TableCell>
              <TableCell align="center" className="num">
                <strong className="points">{row.points}</strong>
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </TableContainer>
  );
}
