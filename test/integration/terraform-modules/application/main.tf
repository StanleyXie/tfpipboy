terraform {
  required_version = ">= 1.0"
}

variable "environment" {
  description = "Environment name"
  type        = string
}

variable "app_name" {
  description = "Application name"
  type        = string
}

variable "subnet_id" {
  description = "Subnet ID where app will be deployed"
  type        = string
}

variable "instance_count" {
  description = "Number of instances"
  type        = number
  default     = 2
}

# Simulate application deployment
resource "null_resource" "application" {
  count = var.instance_count

  triggers = {
    environment    = var.environment
    app_name       = var.app_name
    subnet_id      = var.subnet_id
    instance_index = count.index
    timestamp      = timestamp()
  }

  provisioner "local-exec" {
    command = "echo 'Deploying ${var.app_name} instance ${count.index} in subnet ${var.subnet_id} for ${var.environment}'"
  }
}

output "app_instances" {
  description = "Application instance IDs"
  value = [
    for idx in range(var.instance_count) :
    "instance-${var.environment}-${var.app_name}-${idx}"
  ]
}

output "app_name" {
  description = "Application name"
  value       = var.app_name
}

output "deployment_status" {
  description = "Deployment status"
  value       = "deployed"
}
