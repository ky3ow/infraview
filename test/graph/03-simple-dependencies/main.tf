terraform {
  required_providers {
    tfcoremock = {
      source  = "hashicorp/tfcoremock"
      version = "0.1.2"
    }
  }
}

resource "tfcoremock_simple_resource" "original" {
  float = 2
}

resource "tfcoremock_simple_resource" "implicit" {
  float = tfcoremock_simple_resource.original.float
}

resource "tfcoremock_simple_resource" "explicit" {
  float = 2

  depends_on = [ tfcoremock_simple_resource.original ]
}
