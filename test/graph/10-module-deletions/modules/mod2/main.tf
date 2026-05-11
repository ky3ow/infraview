terraform {
  required_providers {
    tfcoremock = {
      source  = "hashicorp/tfcoremock"
      version = "0.1.2"
    }
  }
}

variable "a" {
  type = bool
}

resource "tfcoremock_simple_resource" "another_simple3" {
  id = "id-sample-3"
  bool = var.a
}

# resource "tfcoremock_simple_resource" "another_simple4" {
#   id = "id-sample-4"
#   bool = tfcoremock_simple_resource.another_simple3.bool
# }
