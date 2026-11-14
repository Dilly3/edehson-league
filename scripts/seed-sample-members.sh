#!/usr/bin/env bash
# Adds five sample members to the LOCAL league (backend on 127.0.0.1:8011).
# Reads ADMIN_TOKEN from .env in the repo root. Run: ./scripts/seed-sample-members.sh
set -euo pipefail
cd "$(dirname "$0")/.."
TOKEN=$(grep '^ADMIN_TOKEN=' .env | cut -d= -f2- | tr -d '"'"'"' \r')
API=${API:-http://127.0.0.1:8011}
while IFS='|' read -r name joined; do
  code=$(curl -s -o /tmp/seed-out -w '%{http_code}' -X POST "$API/api/membership/commands" \
    -H "Content-Type: application/json" -H "Authorization: Bearer $TOKEN" \
    -d "{\"type\":\"add_member\",\"name\":\"$name\",\"joined_on\":\"$joined\"}")
  echo "$name: $code $(cat /tmp/seed-out)"
done <<'NAMES'
Chukwuemeka Okafor|2026-01-10
Tunde Adeyemi|2026-02-03
Ifeanyi Nwosu|2026-03-15
Babajide Ogunleye|2026-05-22
Emeka Eze|2026-08-01
NAMES
