terraform {
  required_providers {
    tfcoremock = {
      source  = "hashicorp/tfcoremock"
      version = "0.1.2"
    }
  }
}

resource "tfcoremock_simple_resource" "gray" {
  float = 5
}

# resource "tfcoremock_simple_resource" "red" {
#   float = tfcoremock_simple_resource.gray.float
# }

# resource "tfcoremock_simple_resource" "red2" {
#   float = 1
#   depends_on = [ tfcoremock_simple_resource.gray ]
# }
