terraform {
  required_providers {
    tfcoremock = {
      source  = "hashicorp/tfcoremock"
      version = "0.1.2"
    }
  }
}

variable "var1_module" {
  type = number
  default = 2
}

variable "var2_module" {
  type = number
}

resource "tfcoremock_simple_resource" "simple" {
  id = "id${var.var1_module}"
  integer = var.var1_module
  float = var.var2_module
}

module "sample_child" {
  source = "./modules"
  var1_nested = var.var1_module
}
