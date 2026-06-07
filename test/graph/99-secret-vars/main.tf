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
  sensitive = true
}

variable "var2_root" {
  type = string
}

resource "tfcoremock_simple_resource" "name" {
  id = "core-id"
  integer = var.var1_root
  string = var.var2_root
}

resource "tfcoremock_simple_resource" "name2" {
  id = "core-id2"
  integer = tfcoremock_simple_resource.name.integer
}

resource "tfcoremock_simple_resource" "name3" {
  id = "core-id3"
  integer = tfcoremock_simple_resource.name.integer
}
