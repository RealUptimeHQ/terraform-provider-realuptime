# Find the page created at signup by its slug, then publish a check on it
# without importing the page into Terraform.
data "realuptime_status_page" "main" {
  slug = "acme"
}

resource "realuptime_status_page_component" "www" {
  status_page_id = data.realuptime_status_page.main.id
  check_id       = realuptime_check.www.id
}
