terraform {
  required_version = ">= 1.0"
}

variable "environment" {
  description = "Environment name"
  type        = string
}

variable "database_name" {
  description = "Database name"
  type        = string
}

variable "subnet_id" {
  description = "Subnet ID where database will be deployed"
  type        = string
}

variable "engine" {
  description = "Database engine"
  type        = string
  default     = "postgres"
}

variable "engine_version" {
  description = "Database engine version"
  type        = string
  default     = "14.0"
}

# Simulate database creation
resource "null_resource" "database" {
  triggers = {
    environment    = var.environment
    database_name  = var.database_name
    subnet_id      = var.subnet_id
    engine         = var.engine
    engine_version = var.engine_version
    timestamp      = timestamp()
  }

  provisioner "local-exec" {
    command = "echo 'Creating ${var.engine} ${var.engine_version} database ${var.database_name} in subnet ${var.subnet_id} for ${var.environment}'"
  }
}

output "database_id" {
  description = "Database instance ID"
  value       = "db-${var.environment}-${var.database_name}-${md5(var.subnet_id)}"
}

output "database_endpoint" {
  description = "Database endpoint"
  value       = "${var.database_name}.${var.environment}.example.com:5432"
}

output "database_status" {
  description = "Database status"
  value       = "available"
}
