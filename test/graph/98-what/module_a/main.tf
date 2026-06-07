terraform {
  required_providers {
    tfcoremock = {
      source  = "hashicorp/tfcoremock"
      version = "0.1.2"
    }
  }
}

variable "var1_child" {
  type = number
  default = 2
}

resource "tfcoremock_simple_resource" "name" {
  id = "child-core-id"
  integer = var.var1_child
}

