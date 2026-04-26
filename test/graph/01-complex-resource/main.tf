terraform {
  required_providers {
    tfcoremock = {
      source  = "hashicorp/tfcoremock"
      version = "0.1.2"
    }
  }
}

resource "tfcoremock_complex_resource" "simple" {
  list = [{string = "1"}, {string = "2"}]
  list_block {
    string = "one"
  }
  list_block {
    string = "two"
  }

  set_block {
    string = "one"
  }

  set_block {
    string = "two"
  }

  map = {
    "one" : {
      string = "map.one"
    },
    "two" : {
      string = "map.two"
    }
  }

  object = {

    string = "nested object"

    object = {
      string = "nested nested object"
    }
  }
}

