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

resource "doppler_secrets_sync_github_actions" "ci" {
  count = local.environment == "ci" ? 1 : 0

  integration     = var.github_integration
  project         = var.project
  config          = local.environment
  sync_target     = "repo"
  repo_name       = var.project
  delete_behavior = "delete_from_target"

  lifecycle {
    precondition {
      condition     = var.github_integration != null
      error_message = "github_integration (the Doppler GitHub integration slug) must be set in terraform.tfvars for the ci workspace."
    }
  }
}
