terraform {
  required_version = ">= 1.0"
}

variable "environment" {
  description = "Environment name"
  type        = string
}

variable "vpc_cidr" {
  description = "VPC CIDR block from which to derive subnet"
  type        = string
}

variable "subnet_suffix" {
  description = "Suffix for subnet (public/private)"
  type        = string
  default     = "public"
}

variable "availability_zone" {
  description = "Availability zone"
  type        = string
  default     = "us-west-2a"
}

# Simulate subnet creation
resource "null_resource" "subnet" {
  triggers = {
    environment       = var.environment
    vpc_cidr          = var.vpc_cidr
    subnet_suffix     = var.subnet_suffix
    availability_zone = var.availability_zone
    timestamp         = timestamp()
  }

  provisioner "local-exec" {
    command = "echo 'Creating ${var.subnet_suffix} subnet in VPC ${var.vpc_cidr} for ${var.environment}'"
  }
}

output "subnet_id" {
  description = "Subnet ID"
  value       = "subnet-${var.environment}-${var.subnet_suffix}-${md5(var.vpc_cidr)}"
}

output "subnet_cidr" {
  description = "Subnet CIDR block"
  value       = cidrsubnet(var.vpc_cidr, 8, 1)
}

output "availability_zone" {
  description = "Availability zone"
  value       = var.availability_zone
}
