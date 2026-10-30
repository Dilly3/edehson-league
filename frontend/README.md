# RealEdehsons League frontend

React, TypeScript, Vite, and Material UI. Connects to the existing Go API; no demo data is inserted.

## Run locally

Start your existing backend stack first. Then, from this folder:

```sh
npm ci
npm run dev
```

Open http://localhost:5173. Requests to `/api` are proxied to http://127.0.0.1:8011. To change the API target, copy `.env.example` to `.env.local` and edit `API_PROXY_TARGET`.

```sh
npm test
npm run build
npm run preview
```

`preview` serves the production build on port 4173 and uses the same local API proxy. For deployment, serve `dist/` with an HTTPS web server that forwards `/api` to Go; Vite is a development server.

## Features

- Public overview, standings, fixtures/results and match events.
- Matchday, status, team and player search filters.
- Team rosters, goals and disciplinary statistics.
- Admin team/player registration and fixture generation.
- Public captain badges, admin player editing, captain assignment, and protected player removal.
- Admin kickoff scheduling, venue editing, complete result entry and corrections.
- Own-goal validation, card-count validation, and stale-revision handling.
- Nigeria-time kickoff entry and display, independent of browser timezone.
- Responsive layout, loading/error/empty states, and manual data refresh.

## Administration

Choose **Admin**, then **Set admin key**, and enter your backend `ADMIN_TOKEN`. The key stays in memory only; reloading clears it. The backend checks authorization on each write. The UI does not claim the key is authenticated until a write succeeds. Never put the key in Vite environment variables, source files, or localStorage.

Writes publish immediately. Result corrections replace the complete event list and use the current revision. A stale revision requires reloading the match; the editor never silently overwrites the latest result. Statistics update after a successful save.

Player creation follows the current backend constraints: names 2–20 characters and jersey numbers 1–100, sent as JSON numbers. Existing roster DTOs with capitalized Go field names are normalized in `src/api.ts`.

## Current API limitations

Roster management uses `PUT /api/teams/:tag/players/:playerID`, `PUT /api/teams/:tag/players/:playerID/captain`, and `DELETE /api/teams/:tag/players/:playerID`. Only admins see these controls. Captain assignments replace the previous captain; successful changes reload the roster and statistics. The backend blocks removal of captains and players linked to match events. Removal is soft deletion, with no restore endpoint currently available. API failures remain visible in the dialog and do not optimistically remove a player.

Public pages and public match details poll every 30 seconds while visible. Live and half-time updates are entered by admins; WebSocket subscriptions are not implemented. Use Refresh for an immediate update. The single-season API determines the current season; no season dates are fabricated.

## Source layout

- `src/App.tsx`: app shell, public views, data loading and navigation.
- `src/components/FixtureDialog.tsx`: match details, scheduling, result editor.
- `src/components/AdminForms.tsx`: admin key, team/player forms, fixture generation.
- `src/components/PlayerManagementDialog.tsx`: editing, captain assignment, removal.
- `src/components/Shared.tsx`: standings table and common visual elements.
- `src/api.ts`: fetch wrapper and existing Go response normalization.
- `src/football.ts`: result validation and timezone conversion.
- `src/football.test.ts`: focused football and timezone tests.
- `src/types.ts`: API types.
- `src/styles.css`: responsive styling.

No backend or root Compose files need to be modified to run this frontend locally.

## Simple card records

Event entry supports goal, own_goal, yellow_card, and red_card. Each player may have up to two yellows and one red per match. Card events do not infer a dismissal or block later events. No second-yellow event is offered. Player profiles show goals, yellow cards, red cards, and own goals. Own goals do not count towards the scorer total.

The frontend reads `red_cards` from the simplified statistics API and falls back to the current `straight_red_cards` name while the backend is being updated. The Go validator and database event constraint must use the same four event types.

Venue is displayed on fixture cards and match details. Admins can set, edit, or clear it in the Venue tab, including after a finalized result. A blank venue sends `null`. Venue and kickoff updates use separate endpoints and separate save buttons. No partial combined update is implied.

Player tags are retained in API normalization for compatibility but are not displayed or searched in the frontend. Players are identified visually by name, team, and jersey number.

The visual theme uses pitch green, cream, and lime accents, with a header light/dark toggle. Light mode is the default; a saved preference overrides it. Yellow and red match-event indicators retain their card colours.

## Architecture

See the [project architecture](../README.md#architecture) for AWS hosting, API request flow, database relationships, Terraform state, deployment, and current operational limits.

Admins select Live, Half-time, or Full-time when publishing. Only full-time (`finalized`) results count toward standings and season statistics. Events are aligned to the player’s home/away team side, and rosters sort numerically by jersey number.
