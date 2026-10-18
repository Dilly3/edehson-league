data "aws_caller_identity" "frontend" {}

data "aws_cloudfront_cache_policy" "static" {
  name = "Managed-CachingOptimized"
}

data "aws_cloudfront_cache_policy" "api" {
  name = "Managed-CachingDisabled"
}

data "aws_cloudfront_origin_request_policy" "api" {
  name = "Managed-AllViewerExceptHostHeader"
}

resource "aws_s3_bucket" "frontend" {
  bucket = "edehsons-frontend-${data.aws_caller_identity.frontend.account_id}-${var.aws_region}"

  # Only replaceable frontend build files belong in this bucket.
  # Allows terraform destroy to remove it even after deployment.
  force_destroy = true
}

resource "aws_s3_bucket_public_access_block" "frontend" {
  bucket = aws_s3_bucket.frontend.id

  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

resource "aws_s3_bucket_server_side_encryption_configuration" "frontend" {
  bucket = aws_s3_bucket.frontend.id

  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm = "AES256"
    }
  }
}

resource "aws_cloudfront_origin_access_control" "frontend" {
  name                              = "edehsons-frontend"
  description                       = "CloudFront access to private frontend files"
  origin_access_control_origin_type = "s3"
  signing_behavior                  = "always"
  signing_protocol                  = "sigv4"
}

resource "aws_cloudfront_distribution" "frontend" {
  enabled             = true
  is_ipv6_enabled     = true
  default_root_object = "index.html"
  comment             = "RealEdehsons League"
  price_class         = "PriceClass_100"
  wait_for_deployment = true

  origin {
    domain_name              = aws_s3_bucket.frontend.bucket_regional_domain_name
    origin_id                = "frontend-s3"
    origin_access_control_id = aws_cloudfront_origin_access_control.frontend.id
  }

  origin {
    domain_name = "edehsons-league.duckdns.org"
    origin_id   = "lightsail-api"

    custom_origin_config {
      http_port              = 80
      https_port             = 443
      origin_protocol_policy = "https-only"
      origin_ssl_protocols   = ["TLSv1.2"]
    }
  }

  default_cache_behavior {
    target_origin_id       = "frontend-s3"
    viewer_protocol_policy = "redirect-to-https"

    allowed_methods = ["GET", "HEAD"]
    cached_methods  = ["GET", "HEAD"]

    compress        = true
    cache_policy_id = data.aws_cloudfront_cache_policy.static.id
  }

  dynamic "ordered_cache_behavior" {
    for_each = ["/api", "/api/*"]

    content {
      path_pattern           = ordered_cache_behavior.value
      target_origin_id       = "lightsail-api"
      viewer_protocol_policy = "https-only"

      allowed_methods = [
        "GET",
        "HEAD",
        "OPTIONS",
        "POST",
        "PUT",
        "PATCH",
        "DELETE"
      ]

      cached_methods = ["GET", "HEAD"]

      compress                 = true
      cache_policy_id          = data.aws_cloudfront_cache_policy.api.id
      origin_request_policy_id = data.aws_cloudfront_origin_request_policy.api.id
    }
  }

  restrictions {
    geo_restriction {
      restriction_type = "none"
    }
  }

  viewer_certificate {
    cloudfront_default_certificate = true
  }
}

resource "aws_s3_bucket_policy" "frontend" {
  bucket = aws_s3_bucket.frontend.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid    = "AllowCloudFrontRead"
        Effect = "Allow"
        Principal = {
          Service = "cloudfront.amazonaws.com"
        }
        Action   = "s3:GetObject"
        Resource = "${aws_s3_bucket.frontend.arn}/*"
        Condition = {
          StringEquals = {
            "AWS:SourceArn" = aws_cloudfront_distribution.frontend.arn
          }
        }
      },
      {
        Sid       = "DenyInsecureTransport"
        Effect    = "Deny"
        Principal = "*"
        Action    = "s3:*"
        Resource = [
          aws_s3_bucket.frontend.arn,
          "${aws_s3_bucket.frontend.arn}/*"
        ]
        Condition = {
          Bool = {
            "aws:SecureTransport" = "false"
          }
        }
      }
    ]
  })

  depends_on = [
    aws_s3_bucket_public_access_block.frontend
  ]
}

output "frontend_bucket_name" {
  value = aws_s3_bucket.frontend.id
}

output "frontend_url" {
  value = "https://${aws_cloudfront_distribution.frontend.domain_name}"
}

output "frontend_distribution_id" {
  value = aws_cloudfront_distribution.frontend.id
}
