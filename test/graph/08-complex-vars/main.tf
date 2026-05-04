terraform {
  required_providers {
    tfcoremock = {
      source  = "hashicorp/tfcoremock"
      version = "0.1.2"
    }
  }
}

variable "var1_root" {
  type = number
  default = 2
}

resource "tfcoremock_simple_resource" "name" {
  id = "core-id"
  integer = 2
}

module "sample" {
  source = "./modules/mod"
  var1_module = var.var1_root
  var2_module = tfcoremock_simple_resource.name.integer
}
