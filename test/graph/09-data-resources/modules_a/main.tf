terraform {
  required_providers {
    tfcoremock = {
      source  = "hashicorp/tfcoremock"
      version = "0.1.2"
    }
  }
}

data "tfcoremock_simple_resource" "external_module" {
  id = "data-id2"
}

resource "tfcoremock_simple_resource" "name_module" {
  id = "module-core-id2"
  integer = data.tfcoremock_simple_resource.external_module.integer
}
