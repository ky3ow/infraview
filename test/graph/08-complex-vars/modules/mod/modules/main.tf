terraform {
  required_providers {
    tfcoremock = {
      source  = "hashicorp/tfcoremock"
      version = "0.1.2"
    }
  }
}

variable "var1_nested" {
  type = number
  default = 2
}

resource "tfcoremock_simple_resource" "simple" {
  id = "nested_id${var.var1_nested}"
  integer = var.var1_nested
}
