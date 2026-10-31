variable "aws_region" {
  type    = string
  default = "eu-west-2"
}

variable "availability_zone" {
  type    = string
  default = "eu-west-2a"
}

variable "ssh_cidr" {
  description = "Your public IPv4 address followed by /32."
  type        = string

  validation {
    condition = (
      can(cidrnetmask(var.ssh_cidr)) &&
      endswith(var.ssh_cidr, "/32")
    )
    error_message = "Supply a single IPv4 address, such as 203.0.113.10/32."
  }
}

variable "ssh_public_key_path" {
  description = "Path to your SSH public key. Never supply the private key."
  type        = string
}

variable "ssh_allowed_cidrs" {
  description = "Operator IPv4 CIDRs allowed to SSH; unrelated to application login rate limits."
  type        = list(string)
  validation {
    condition = length(var.ssh_allowed_cidrs) > 0 && alltrue([
      for cidr in var.ssh_allowed_cidrs : can(cidrnetmask(cidr)) && try(tonumber(split("/", cidr)[1]) >= 16, false)
    ])
    error_message = "Use IPv4 CIDRs with /16 or narrower masks; prefer your current public IP/32."
  }
}
