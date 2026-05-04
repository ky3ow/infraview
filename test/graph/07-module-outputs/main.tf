terraform {
  required_providers {
    tfcoremock = {
      source  = "hashicorp/tfcoremock"
      version = "0.1.2"
    }
  }
}

module "sample" {
  source = "./modules/mod"
}

output "outer" {
  value = module.sample.two_passthrough_output
}
