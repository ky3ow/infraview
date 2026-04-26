terraform {
  required_providers {
    tfcoremock = {
      source  = "hashicorp/tfcoremock"
      version = "0.1.2"
    }
  }
}

resource "tfcoremock_simple_resource" "simple" {
  id = "id"
  string = "string"
  integer = 3
  float = 2
  bool = true
}
