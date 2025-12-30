terraform {
  required_providers {
    tfcoremock = {
      source  = "hashicorp/tfcoremock"
      version = "0.1.2"
    }
  }
}

resource "tfcoremock_simple_resource" "example" {
  id      = "my-simple-resource"
  bool    = true
  number  = 2
  string  = "Hello, world!"
  float   = 0
  integer = 0
}

resource "tfcoremock_simple_resource" "example_dep" {
  id      = "my-simple-resource-2"
  bool    = true
  number  = tfcoremock_simple_resource.example.number + 3
  string  = "Hello, world!"
  float   = tfcoremock_simple_resource.example.float + 1.5
  integer = 0
}

resource "tfcoremock_simple_resource" "example_dep_explicit" {
  id      = "my-simple-resource-10"
  bool    = true
  number  = 3
  string  = "Hello, world!"
  float   = 1.5
  integer = 0
  depends_on = [ tfcoremock_simple_resource.example, tfcoremock_simple_resource.example_dep ]
}

resource "tfcoremock_complex_resource" "example" {
  id = "my-complex-resource"

  list = [
    {
      string = tfcoremock_simple_resource.example_dep_explicit.string
    },
    {
      string = "list.two"
    }
  ]

  set = [
    {
      string = "set.one"
    },
    {
      string = tfcoremock_simple_resource.example.string
    }
  ]

  map = {
    "one" : {
      string = "map.one"
    },
    "two" : {
      string = tfcoremock_simple_resource.example.string
    }
  }

  object = {

    string = tfcoremock_simple_resource.example.string

    object = {
      string = "nested nested object"
    }
  }

  list_block {
    string = tfcoremock_simple_resource.example_dep_explicit.string
  }

  list_block {
    string = "list_block.2"
  }

  list_block {
    string = "list_block.3"
  }

  set_block {
    string = tfcoremock_simple_resource.example_dep_explicit.string
  }

  set_block {
    string = "set_block.two"
  }
}

module "example_module" {
  source = "./modules/example-module"
  id = "module-id-"
}

module "example_module2" {
  source = "./modules/example-module"
  id = "another-module-id-${module.example_module.num}"
}
