terraform {
  required_providers {
    tfcoremock = {
      source  = "hashicorp/tfcoremock"
      version = "0.1.2"
    }
  }
}

variable "a" {
  default = {
    one = { name = "first" }
    two = { name = "second" }
  }
  type = map(object({
    name = string
  }))
}


resource "tfcoremock_simple_resource" "counted" {
  for_each = var.a
  string = each.value.name
}

resource "tfcoremock_simple_resource" "dep" {
  string = tfcoremock_simple_resource.counted["one"].string
}
