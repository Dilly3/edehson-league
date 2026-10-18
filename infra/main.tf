resource "aws_lightsail_key_pair" "deploy" {
  name       = "edehsons-deploy"
  public_key = trimspace(file(pathexpand(var.ssh_public_key_path)))
}

resource "aws_lightsail_instance" "api" {
  name              = "edehsons-api"
  availability_zone = var.availability_zone

  blueprint_id = "ubuntu_24_04"
  bundle_id    = "nano_3_0"

  key_pair_name = aws_lightsail_key_pair.deploy.name
}

resource "aws_lightsail_static_ip" "api" {
  name = "edehsons-api-ip"
}

resource "aws_lightsail_static_ip_attachment" "api" {
  static_ip_name = aws_lightsail_static_ip.api.name
  instance_name  = aws_lightsail_instance.api.name
}

resource "aws_lightsail_instance_public_ports" "api" {
  instance_name = aws_lightsail_instance.api.name

  port_info {
    protocol  = "tcp"
    from_port = 22
    to_port   = 22
    cidrs     = [var.ssh_cidr]
  }

  port_info {
    protocol  = "tcp"
    from_port = 80
    to_port   = 80
    cidrs     = ["0.0.0.0/0"]
  }

  port_info {
    protocol  = "tcp"
    from_port = 443
    to_port   = 443
    cidrs     = ["0.0.0.0/0"]
  }
}
