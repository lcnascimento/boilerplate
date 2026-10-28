variable "project" {
  type    = string
  default = "boilerplate"
}

variable "state_passphrase" {
  type      = string
  sensitive = true
}
