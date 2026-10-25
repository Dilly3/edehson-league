# CloudFront viewer certificates must be created in us-east-1.
provider "aws" {
  alias  = "cloudfront_certificate"
  region = "us-east-1"

  default_tags {
    tags = {
      Project   = "edehsons-league"
      ManagedBy = "terraform"
    }
  }
}

resource "aws_acm_certificate" "frontend" {
  provider          = aws.cloudfront_certificate
  domain_name       = "edehsons.sportslive.com.ng"
  validation_method = "DNS"

  lifecycle {
    create_before_destroy = true
  }
}

output "frontend_certificate_arn" {
  value = aws_acm_certificate.frontend.arn
}

# GO54 manages DNS. Add these records there rather than creating a Route 53 zone.
output "frontend_certificate_dns_records" {
  value = [for record in aws_acm_certificate.frontend.domain_validation_options : {
    name  = record.resource_record_name
    type  = record.resource_record_type
    value = record.resource_record_value
  }]
}

# DNS validation records are kept at GO54 for automatic renewal.
resource "aws_acm_certificate_validation" "frontend" {
  provider                = aws.cloudfront_certificate
  certificate_arn         = aws_acm_certificate.frontend.arn
  validation_record_fqdns = [for record in aws_acm_certificate.frontend.domain_validation_options : record.resource_record_name]
}

# Keep the active certificate during the GreatMindsSport domain transition.
resource "aws_acm_certificate" "greatminds" {
  provider                  = aws.cloudfront_certificate
  domain_name               = "greatminds.sportslive.com.ng"
  subject_alternative_names = ["edehsons.sportslive.com.ng"]
  validation_method         = "DNS"
  lifecycle { create_before_destroy = true }
}
resource "aws_acm_certificate_validation" "greatminds" {
  provider                = aws.cloudfront_certificate
  certificate_arn         = aws_acm_certificate.greatminds.arn
  validation_record_fqdns = [for record in aws_acm_certificate.greatminds.domain_validation_options : record.resource_record_name]
}
output "greatminds_certificate_dns_records" {
  value = [for record in aws_acm_certificate.greatminds.domain_validation_options : {
    name  = record.resource_record_name
    type  = record.resource_record_type
    value = record.resource_record_value
  }]
}
