#!/usr/bin/env bash
set -Eeuo pipefail

# Run on your Mac. Deploys ONLY the frontend.
cd "$(dirname -- "${BASH_SOURCE[0]}")"
export AWS_PROFILE="${AWS_PROFILE:-fambond-new-dilly3}"

# Fail early with a clear message if the AWS sign-in has expired.
if ! aws sts get-caller-identity >/dev/null 2>&1; then
  echo "AWS credentials for profile '$AWS_PROFILE' are missing or expired." >&2
  echo "Sign in, then run this script again:  aws sso login --profile $AWS_PROFILE" >&2
  exit 1
fi

npm --prefix frontend run build

FRONTEND_BUCKET=$(terraform -chdir=infra output -raw frontend_bucket_name)
DISTRIBUTION_ID=$(terraform -chdir=infra output -raw frontend_distribution_id)

aws s3 sync frontend/dist/ "s3://$FRONTEND_BUCKET/" \
  --exclude "index.html" \
  --exclude "version.json" \
  --cache-control "public,max-age=3600"

aws s3 cp frontend/dist/index.html \
  "s3://$FRONTEND_BUCKET/index.html" \
  --content-type "text/html" \
  --cache-control "no-cache,max-age=0,must-revalidate"

# Open tabs poll this to detect a new deploy, so it must never be cached.
aws s3 cp frontend/dist/version.json \
  "s3://$FRONTEND_BUCKET/version.json" \
  --content-type "application/json" \
  --cache-control "no-cache,max-age=0,must-revalidate"

aws cloudfront create-invalidation \
  --distribution-id "$DISTRIBUTION_ID" \
  --paths "/" "/index.html" "/version.json" "/crest.svg" "/admin" "/admin/"