terraform {
  required_version = ">= 1.0"
}

variable "environment" {
  description = "Environment name"
  type        = string
}

variable "cidr_block" {
  description = "CIDR block for VPC"
  type        = string
  default     = "10.0.0.0/16"
}

variable "region" {
  description = "AWS region"
  type        = string
  default     = "us-west-2"
}

# Simulate VPC creation
resource "null_resource" "vpc" {
  triggers = {
    environment = var.environment
    cidr_block  = var.cidr_block
    timestamp   = timestamp()
  }

  provisioner "local-exec" {
    command = "echo 'Creating VPC for ${var.environment} with CIDR ${var.cidr_block}'"
  }
}

output "vpc_id" {
  description = "VPC ID"
  value       = "vpc-${var.environment}-${md5(var.cidr_block)}"
}

output "vpc_cidr" {
  description = "VPC CIDR block"
  value       = var.cidr_block
}

output "environment" {
  description = "Environment"
  value       = var.environment
}
