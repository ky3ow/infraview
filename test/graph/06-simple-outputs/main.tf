terraform {
  required_providers {
    tfcoremock = {
      source  = "hashicorp/tfcoremock"
      version = "0.1.2"
    }
  }
}

resource "tfcoremock_simple_resource" "simple" {
  id = "id"
  integer = 2
}

output "simple_int" {
  value = tfcoremock_simple_resource.simple.integer
  description = "simple description"
}
