terraform {
  required_providers {
    tfcoremock = {
      source  = "hashicorp/tfcoremock"
      version = "0.1.2"
    }
  }
}
resource "tfcoremock_simple_resource" "simple" {
  id = "id_one"
}

output "one_output" {
  value = tfcoremock_simple_resource.simple.id
}

module "sample_child" {
  source = "./modules"
}

output "two_passthrough_output" {
  value = module.sample_child.two_output
}
