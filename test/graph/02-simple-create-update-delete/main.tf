terraform {
  required_providers {
    tfcoremock = {
      source  = "hashicorp/tfcoremock"
      version = "0.1.2"
    }
  }
}

resource "tfcoremock_simple_resource" "green" {
  float = 2
}

resource "tfcoremock_simple_resource" "yellow" {
  float = 5
}

# resource "tfcoremock_simple_resource" "red" {
#   float = 2
# }

