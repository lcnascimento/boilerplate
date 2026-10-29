terraform {
  required_providers {
    doppler = {
      source = "DopplerHQ/doppler"
    }
  }
}

resource "doppler_project" "this" {
  name = var.project
}

resource "doppler_environment" "this" {
  for_each = var.environments

  project          = doppler_project.this.name
  slug             = each.key
  name             = each.value
  personal_configs = false
}

resource "doppler_service_token" "dev" {
  for_each = var.dev_token_holders

  project = doppler_project.this.name
  config  = doppler_environment.this["dev"].slug
  name    = each.key
  access  = "read"
}

resource "doppler_secrets_sync_github_actions" "ci" {
  count = var.github_integration == null ? 0 : 1

  integration     = var.github_integration
  project         = doppler_project.this.name
  config          = doppler_environment.this["ci"].slug
  sync_target     = "repo"
  repo_name       = var.project
  delete_behavior = "delete_from_target"
}
