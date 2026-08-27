# Publish the API check on the status page under a public label.
resource "realuptime_status_page_component" "api" {
  status_page_id = realuptime_status_page.public.id
  check_id       = realuptime_check.api.id
  display_name   = "API"
  description    = "REST API and webhooks"
}
