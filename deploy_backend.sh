#!/usr/bin/env bash
set -Eeuo pipefail
umask 077

# Run on your Mac. This script deploys ONLY the backend.
PROJECT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
DEPLOY_HOST="${DEPLOY_HOST:-ubuntu@13.134.164.31}"
SSH_KEY="${SSH_KEY:-$HOME/.ssh/edehsons}"
SSH_OPTIONS=(-i "$SSH_KEY" -o BatchMode=yes -o ConnectTimeout=15)
RELEASE_ID="$(date -u +%Y%m%dT%H%M%SZ)-$$"
IMAGE="edehson-backend:$RELEASE_ID"
ARCHIVE="edehson-backend-$RELEASE_ID.tar.gz"
cd "$PROJECT_DIR"

for dependency in go docker ssh scp gzip; do
  command -v "$dependency" >/dev/null || { echo "Missing command: $dependency" >&2; exit 1; }
done
[[ -f "$SSH_KEY" ]] || { echo "SSH key not found: $SSH_KEY" >&2; exit 1; }
docker info >/dev/null
ssh "${SSH_OPTIONS[@]}" "$DEPLOY_HOST" 'test -d ~/edehson-league && sudo -n docker info >/dev/null'

echo "Running backend tests..."
go -C backend test ./...

echo "Building linux/amd64 release $RELEASE_ID..."
docker buildx build --platform linux/amd64 --load --tag "$IMAGE" ./backend
mkdir -p releases
docker image save "$IMAGE" | gzip > "releases/$ARCHIVE"

echo "Uploading release..."
scp "${SSH_OPTIONS[@]}" "releases/$ARCHIVE" "$DEPLOY_HOST:edehson-league/$ARCHIVE"

ssh "${SSH_OPTIONS[@]}" "$DEPLOY_HOST" bash -s -- "$RELEASE_ID" <<'REMOTE'
set -Eeuo pipefail
umask 077
release_id="$1"
cd "$HOME/edehson-league"
exec 9>.backend-deploy.lock
flock -n 9 || { echo "Another backend deployment is running." >&2; exit 1; }

image="edehson-backend:$release_id"
archive="edehson-backend-$release_id.tar.gz"
rollback_image="edehson-backend:rollback-$release_id"
compose() {
  sudo -n docker compose --env-file .env.production -p edehson-prod -f compose.production.yml "$@"
}
compose config --quiet
# The production configuration currently uses this stable image tag.
configured_image="$(compose config --format json | python3 -c 'import json,sys; print(json.load(sys.stdin)["services"]["backend"]["image"])')"
if [[ "$configured_image" != "edehson-backend:release" ]]; then
  echo "Expected BACKEND_IMAGE=edehson-backend:release; stopping before changes." >&2
  exit 1
fi
container="$(compose ps -q backend)"
[[ -n "$container" ]] || { echo "No existing backend container; use initial setup first." >&2; exit 1; }
old_image="$(sudo -n docker inspect --format '{{.Image}}' "$container")"
sudo -n docker image tag "$old_image" "$rollback_image"

mkdir -p backups
chmod 700 backups
backup="backups/pre-deploy-$release_id.dump"
echo "Backing up PostgreSQL..."
compose exec -T postgres sh -c 'pg_dump -U "$POSTGRES_USER" -d "$POSTGRES_DB" -Fc' < /dev/null > "$backup.partial"
[[ -s "$backup.partial" ]]
# Check that the archive is readable; this is not a full restore test.
compose exec -T postgres pg_restore --list < "$backup.partial" > /dev/null
mv "$backup.partial" "$backup"
echo "Backup saved: $HOME/edehson-league/$backup"

sudo -n docker image load --input "$archive"
[[ "$(sudo -n docker image inspect --format '{{.Os}}/{{.Architecture}}' "$image")" == "linux/amd64" ]]

healthy() {
  curl --fail --silent --show-error --connect-timeout 3 --max-time 5 \
    https://edehsons-league.duckdns.org/api/ready 2>/dev/null |
    python3 -c 'import json,sys; d=json.load(sys.stdin); sys.exit(0 if d.get("status")=="ready" and d.get("database")=="connected" else 1)' 2>/dev/null
}
rollback() {
  local status=$?
  trap - ERR
  echo "Deployment failed; restoring the previous backend image..." >&2
  sudo -n docker image tag "$rollback_image" edehson-backend:release || true
  compose up -d --no-deps --force-recreate backend || true
  echo "Database was not restored automatically. Backup: $backup" >&2
  exit "$status"
}
trap rollback ERR
sudo -n docker image tag "$image" edehson-backend:release
compose up -d --no-deps --force-recreate backend

ready=false
for attempt in {1..12}; do
  running_image="$(sudo -n docker inspect --format '{{.Image}}' "$(compose ps -q backend)")"
  expected_image="$(sudo -n docker image inspect --format '{{.Id}}' "$image")"
  if [[ "$running_image" == "$expected_image" ]] && healthy; then
    ready=true
    break
  fi
  sleep 5
done
if [[ "$ready" != true ]]; then
  compose logs --tail=60 backend >&2 || true
  false
fi
trap - ERR
echo "Backend deployed successfully: $image"
echo "Previous image retained: $rollback_image"
REMOTE

echo "Copying the pre-deployment database backup to your Mac..."
mkdir -p backups
scp "${SSH_OPTIONS[@]}" "$DEPLOY_HOST:edehson-league/backups/pre-deploy-$RELEASE_ID.dump" "backups/"
echo "Done. Frontend and Terraform were not deployed."
