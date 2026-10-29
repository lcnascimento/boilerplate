output "dev_tokens" {
  value     = { for holder, token in doppler_service_token.dev : holder => token.key }
  sensitive = true
}
