cd ~/Desktop/edehson-league
npm --prefix frontend run build

FRONTEND_BUCKET=$(terraform -chdir=infra output -raw frontend_bucket_name)
DISTRIBUTION_ID=$(terraform -chdir=infra output -raw frontend_distribution_id)

aws s3 sync frontend/dist/ "s3://$FRONTEND_BUCKET/" \
  --exclude "index.html" \
  --cache-control "public,max-age=3600"

aws s3 cp frontend/dist/index.html \
  "s3://$FRONTEND_BUCKET/index.html" \
  --content-type "text/html" \
  --cache-control "no-cache,max-age=0,must-revalidate"

aws cloudfront create-invalidation \
  --distribution-id "$DISTRIBUTION_ID" \
  --paths "/" "/index.html" "/crest.svg"