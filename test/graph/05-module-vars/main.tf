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

variable "var2_root" {
  type = number
}

module "sample" {
  source = "./modules/mod"
  var1_module = var.var1_root
  var2_module = var.var2_root
}
