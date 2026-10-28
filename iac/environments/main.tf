terraform {
  required_version = "~> 1.13"

  required_providers {
    doppler = {
      source  = "DopplerHQ/doppler"
      version = "~> 1.22"
    }
  }

  backend "local" {}

  encryption {
    key_provider "pbkdf2" "state" {
      passphrase = var.state_passphrase
    }

    method "aes_gcm" "state" {
      keys = key_provider.pbkdf2.state
    }

    state {
      method   = method.aes_gcm.state
      enforced = true
    }

    plan {
      method   = method.aes_gcm.state
      enforced = true
    }
  }
}

provider "doppler" {}

locals {
  environment = terraform.workspace
}

resource "doppler_service_token" "dev" {
  for_each = local.environment == "dev" ? var.dev_token_holders : toset([])

  project = var.project
  config  = local.environment
  name    = each.key
  access  = "read"
}
