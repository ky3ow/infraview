terraform {
  required_providers {
    tfcoremock = {
      source  = "hashicorp/tfcoremock"
      version = "0.1.2"
    }
  }
}

resource "tfcoremock_simple_resource" "simple" {
  id = "nested_id_two"
}

output "two_output" {
  value = tfcoremock_simple_resource.simple
}
