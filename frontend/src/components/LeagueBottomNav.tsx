import type { ReactNode } from "react";
import HomeOutlined from "@mui/icons-material/HomeOutlined";
import LeaderboardOutlined from "@mui/icons-material/LeaderboardOutlined";
import CalendarMonthOutlined from "@mui/icons-material/CalendarMonthOutlined";
import PersonOutlined from "@mui/icons-material/PersonOutlined";
import GroupsOutlined from "@mui/icons-material/GroupsOutlined";
import BottomNav from "./BottomNav";
import { pageURL, type Page } from "../navigation";

const icons: Partial<Record<Page, ReactNode>> = {
  overview: <HomeOutlined />,
  standings: <LeaderboardOutlined />,
  fixtures: <CalendarMonthOutlined />,
  players: <PersonOutlined />,
  teams: <GroupsOutlined />,
};

export default function LeagueBottomNav({ items, page, adminArea, onNavigate }: {
  items: { id: Page; label: string }[];
  page: Page;
  adminArea: boolean;
  onNavigate: (page: Page) => void;
}) {
  return <BottomNav label="League navigation" items={items.map(({ id, label }) => ({
    key: id, label, icon: icons[id], active: page === id, href: pageURL(id, adminArea), onSelect: () => onNavigate(id),
  }))} />;
}
