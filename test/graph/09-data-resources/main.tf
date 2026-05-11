terraform {
  required_providers {
    tfcoremock = {
      source  = "hashicorp/tfcoremock"
      version = "0.1.2"
    }
  }
}

data "tfcoremock_simple_resource" "external" {
  id = "data-id"
}

resource "tfcoremock_simple_resource" "name" {
  id = "core-id"
  integer = data.tfcoremock_simple_resource.external.integer
}

module "test" {
  source = "./modules_a"
}
