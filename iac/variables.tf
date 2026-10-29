variable "project" {
  type    = string
  default = "boilerplate"
}

variable "state_passphrase" {
  type      = string
  sensitive = true
}

variable "dev_token_holders" {
  type    = set(string)
  default = []
}

variable "github_integration" {
  type    = string
  default = null
}
