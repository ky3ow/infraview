terraform {
  required_providers {
    tfcoremock = {
      source  = "hashicorp/tfcoremock"
      version = "0.1.2"
    }
  }
}

variable "var1" {
  type = number
  default = 2
}

variable "var2" {
  type = number
}

resource "tfcoremock_simple_resource" "simple" {
  id = "id"
  integer = var.var1
  float = var.var2
}
