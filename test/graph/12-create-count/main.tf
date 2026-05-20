terraform {
  required_providers {
    tfcoremock = {
      source  = "hashicorp/tfcoremock"
      version = "0.1.2"
    }
  }
}

variable "a" {
  default = 3
  type = number
}


resource "tfcoremock_simple_resource" "counted" {
  count = var.a
  integer = count.index
}

resource "tfcoremock_simple_resource" "dep" {
  integer = tfcoremock_simple_resource.counted.1.integer
}

# resource "tfcoremock_simple_resource" "red2" {
#   float = 1
#   depends_on = [ tfcoremock_simple_resource.gray ]
# }
