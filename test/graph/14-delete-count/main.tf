terraform {
  required_providers {
    tfcoremock = {
      source  = "hashicorp/tfcoremock"
      version = "0.1.2"
    }
  }
}

variable "a" {
  default = 0
  type = number
}


# resource "tfcoremock_simple_resource" "counted" {
#   count = var.a
#   integer = count.index
# }

resource "tfcoremock_simple_resource" "plus" {
  count = var.a + 1
  integer = count.index
}

# resource "tfcoremock_simple_resource" "dep" {
#   integer = tfcoremock_simple_resource.counted.0.integer
# }
