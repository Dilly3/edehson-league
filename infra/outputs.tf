output "public_ip" {
  value = aws_lightsail_static_ip.api.ip_address
}

output "api_domain" {
  value = "edehsons-league.duckdns.org"
}
