variable "discord_webhook_url" {
  type      = string
  sensitive = true
}

resource "realuptime_notification_channel" "discord" {
  type = "discord"
  name = "Ops Discord"
  config = {
    webhookUrl = var.discord_webhook_url
  }
}

resource "realuptime_notification_channel" "opsgenie" {
  type = "opsgenie"
  name = "Opsgenie (EU)"
  config = {
    apiKey = var.opsgenie_api_key
    region = "eu"
  }
}

resource "realuptime_notification_channel" "zapier" {
  type    = "generic_json"
  name    = "Zapier hook"
  enabled = false
  config = {
    url = "https://hooks.zapier.com/hooks/catch/123/abc"
  }
}

variable "opsgenie_api_key" {
  type      = string
  sensitive = true
}
