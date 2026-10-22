terraform {
  required_version = ">= 1.10.0, < 2.0.0"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 6.0"
    }
  }
  backend "s3" {}
}

provider "aws" {
  region              = "eu-west-1"
  allowed_account_ids = ["821643761950"]
  default_tags {
    tags = {
      Project     = "edehsons-league"
      Environment = "streaming-test"
      ManagedBy   = "terraform"
    }
  }
}

# BASIC passes through the source quality; suitable for an initial phone test.
# No recording configuration means automatic recording is disabled.
resource "aws_ivs_channel" "test" {
  name         = "edehsons-streaming-test"
  type         = "BASIC"
  latency_mode = "LOW"
  authorized   = false
}

data "aws_ivs_stream_key" "test" {
  channel_arn = aws_ivs_channel.test.arn
}

output "channel_arn" {
  value = aws_ivs_channel.test.arn
}

output "playback_url" {
  value = aws_ivs_channel.test.playback_url
}

output "larix_connection_url" {
  value     = "rtmps://${aws_ivs_channel.test.ingest_endpoint}:443/app/${data.aws_ivs_stream_key.test.value}"
  sensitive = true
}
