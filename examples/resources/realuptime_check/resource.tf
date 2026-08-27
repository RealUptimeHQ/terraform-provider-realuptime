# An http monitor probed from two regions every 60 seconds, failing unless
# the response is a 2xx whose body contains "ok".
resource "realuptime_check" "api" {
  name             = "Public API"
  url              = "https://api.example.com/health"
  interval_seconds = 60
  regions          = ["iad", "fra"]

  assertion_status_min = 200
  assertion_status_max = 299
  assertion_body_op    = "contains"
  assertion_body_value = "ok"
}

# Minimal: every live region, the account's default interval, no assertions.
resource "realuptime_check" "www" {
  name = "Website"
  url  = "https://www.example.com/"
}
