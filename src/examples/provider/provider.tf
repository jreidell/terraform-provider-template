terraform {
  required_providers {
    template = {
      source = "jreidell/template"
    }
  }
}

provider "template" {
  # Optional. Falls back to TEMPLATE_ENDPOINT, then to the public uuidtools.com API.
  # endpoint = "https://www.uuidtools.com/api/generate/v1"
}
