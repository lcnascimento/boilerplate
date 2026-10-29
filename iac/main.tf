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
  environments = {
    dev = { name = "Development", created_as = "dev" }
    ci  = { name = "CI", created_as = "stg" }
    prd = { name = "Production", created_as = "prd" }
  }
}

import {
  to = module.doppler.doppler_project.this
  id = var.project
}

import {
  for_each = local.environments
  to       = module.doppler.doppler_environment.this[each.key]
  id       = "${var.project}.${each.value.created_as}"
}

module "doppler" {
  source             = "./modules/doppler"
  project            = var.project
  environments       = { for slug, environment in local.environments : slug => environment.name }
  dev_token_holders  = var.dev_token_holders
  github_integration = var.github_integration
}
