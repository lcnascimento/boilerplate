variable "project" {
  type = string
}

variable "environments" {
  type = map(string)
}

variable "dev_token_holders" {
  type = set(string)
}

variable "github_integration" {
  type = string
}
