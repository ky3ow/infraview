terraform {
  required_providers {
    tfcoremock = {
      source  = "hashicorp/tfcoremock"
      version = "0.1.2"
    }
  }
}

variable "b" {
  type = bool
}

resource "tfcoremock_simple_resource" "simple" {
  id = "id-sample-1"
  bool = var.b
}

module "sample_child" {
  source = "./modules"
}
