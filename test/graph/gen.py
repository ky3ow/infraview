import sys

with open(f"{sys.argv[1]}/main.tf", "w") as file:
    result = """
terraform {
  required_providers {
    tfcoremock = {
      source  = "hashicorp/tfcoremock"
      version = "0.1.2"
    }
  }
}

resource "tfcoremock_simple_resource" "name0" {
integer = 0
}\n"""
    file.write(result)
    for start in range(1, int(sys.argv[2])):
        result = f"""resource "tfcoremock_simple_resource" "name{start}" {{
integer = tfcoremock_simple_resource.name{start - 1}.integer + 1
}}\n"""
        file.write(result)


