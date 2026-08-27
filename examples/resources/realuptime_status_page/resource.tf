resource "realuptime_status_page" "public" {
  name             = "Acme Status"
  slug             = "acme"
  page_title       = "Acme platform status"
  page_description = "Live status of the Acme API, dashboard and webhooks."
  accent_color     = "#1f6feb"
  theme_preset     = "slate"
  locale           = "en-GB"
  homepage_url     = "https://www.acme.example"
  support_url      = "https://support.acme.example"
  hide_branding    = true # honoured on Growth/Scale only
}
