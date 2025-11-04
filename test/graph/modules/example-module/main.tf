terraform {
  required_providers {
    tfcoremock = {
      source  = "hashicorp/tfcoremock"
      version = "0.1.2"
    }
  }
}

variable "id" {
  type = string
}

resource "tfcoremock_simple_resource" "example" {
  id = var.id
}

resource "tfcoremock_simple_resource" "example2" {
  id = "${tfcoremock_simple_resource.example.id}-another"
  number = 3
}

output "num" {
  value = tfcoremock_simple_resource.example2.number
}
