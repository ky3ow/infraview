terraform {
  required_providers {
    tfcoremock = {
      source  = "hashicorp/tfcoremock"
      version = "0.1.2"
    }
  }
}

# module "sample" {
#   source = "./modules/mod"
#   b = true
# }

module "sample2" {
  source = "./modules/mod2"
  a = false
}
