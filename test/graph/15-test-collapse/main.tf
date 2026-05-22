terraform {
  required_providers {
    tfcoremock = {
      source  = "hashicorp/tfcoremock"
      version = "0.1.2"
    }
  }
}

# resource "tfcoremock_simple_resource" "one" {
#   integer = 1
# }

# resource "tfcoremock_simple_resource" "two" {
#   integer = tfcoremock_simple_resource.one.integer + 1
# }

# resource "tfcoremock_simple_resource" "three" {
#   integer = tfcoremock_simple_resource.two.integer + 1
# }

# resource "tfcoremock_simple_resource" "four" {
#   integer = tfcoremock_simple_resource.three.integer + 1
# }

# resource "tfcoremock_simple_resource" "also_four" {
#   integer = tfcoremock_simple_resource.three.integer + 1
# }
