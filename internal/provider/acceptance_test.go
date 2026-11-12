package provider

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// Acceptance tests run real plans against a live API and need:
//
//	TF_ACC=1                      (terraform-plugin-testing's own gate)
//	REALUPTIME_API_KEY=ru_live_... a read_write key
//	REALUPTIME_API_URL=...        optional; defaults to production. For a
//	                              local dev server: http://localhost:3000/api/v1
//	REALUPTIME_ACC_CHECK_ID=...   optional; an existing http check id to
//	                              publish as a component. Without it the
//	                              component test creates its own check.
//
// and a `terraform` binary on PATH. They create and destroy real resources
// on that account, each named with a timestamp so a failed run leaves
// identifiable litter, never something that looks like a customer's.

var testProtoV6Factories = map[string]func() (tfprotov6.ProviderServer, error){
	"realuptime": providerserver.NewProtocol6WithError(New("acc")()),
}

func accPreCheck(t *testing.T) {
	t.Helper()
	if os.Getenv("REALUPTIME_API_KEY") == "" {
		t.Skip("REALUPTIME_API_KEY not set; skipping acceptance test")
	}
}

func accSuffix() string { return fmt.Sprintf("%d", time.Now().Unix()%1_000_000) }

func TestAccCheck_basic(t *testing.T) {
	suffix := accSuffix()
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { accPreCheck(t) },
		ProtoV6ProviderFactories: testProtoV6Factories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "realuptime_check" "acc" {
  name    = "tf-acc-%s"
  url     = "https://example.com/"
  regions = ["iad", "fra"]
  assertion_status_min = 200
  assertion_status_max = 299
}`, suffix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("realuptime_check.acc", "id"),
					resource.TestCheckResourceAttr("realuptime_check.acc", "type", "http"),
					resource.TestCheckResourceAttr("realuptime_check.acc", "regions.#", "2"),
					resource.TestCheckResourceAttr("realuptime_check.acc", "assertion_status_min", "200"),
				),
			},
			{
				Config: fmt.Sprintf(`
resource "realuptime_check" "acc" {
  name    = "tf-acc-%s"
  url     = "https://example.com/"
  regions = ["iad", "fra", "nrt"]
}`, suffix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("realuptime_check.acc", "regions.#", "3"),
					resource.TestCheckNoResourceAttr("realuptime_check.acc", "assertion_status_min"),
				),
			},
			{
				ResourceName:      "realuptime_check.acc",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccStatusPage_withComponentAndDataSource(t *testing.T) {
	suffix := accSuffix()
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { accPreCheck(t) },
		ProtoV6ProviderFactories: testProtoV6Factories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "realuptime_status_page" "acc" {
  name             = "tf-acc-%[1]s"
  slug             = "tf-acc-%[1]s"
  page_description = "acceptance"
  accent_color     = "#1f6feb"
  theme_preset     = "slate"
}

resource "realuptime_check" "acc" {
  name = "tf-acc-comp-%[1]s"
  url  = "https://example.com/"
}

resource "realuptime_status_page_component" "acc" {
  status_page_id = realuptime_status_page.acc.id
  check_id       = realuptime_check.acc.id
  description    = "tooltip"
}

data "realuptime_status_page" "by_slug" {
  slug = realuptime_status_page.acc.slug
}`, suffix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("realuptime_status_page.acc", "slug", "tf-acc-"+suffix),
					resource.TestCheckResourceAttr("realuptime_status_page.acc", "theme_preset", "slate"),
					resource.TestCheckResourceAttr("realuptime_status_page_component.acc", "display_name", "tf-acc-comp-"+suffix),
					resource.TestCheckResourceAttrPair("data.realuptime_status_page.by_slug", "id", "realuptime_status_page.acc", "id"),
				),
			},
			{
				Config: fmt.Sprintf(`
resource "realuptime_status_page" "acc" {
  name = "tf-acc-%[1]s renamed"
  slug = "tf-acc-%[1]s"
}

resource "realuptime_check" "acc" {
  name = "tf-acc-comp-%[1]s"
  url  = "https://example.com/"
}

resource "realuptime_status_page_component" "acc" {
  status_page_id = realuptime_status_page.acc.id
  check_id       = realuptime_check.acc.id
  display_name   = "Public API"
}`, suffix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckNoResourceAttr("realuptime_status_page.acc", "page_description"),
					resource.TestCheckResourceAttr("realuptime_status_page_component.acc", "display_name", "Public API"),
					resource.TestCheckNoResourceAttr("realuptime_status_page_component.acc", "description"),
				),
			},
			{
				ResourceName:      "realuptime_status_page.acc",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccNotificationChannel_basic(t *testing.T) {
	suffix := accSuffix()
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { accPreCheck(t) },
		ProtoV6ProviderFactories: testProtoV6Factories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "realuptime_notification_channel" "acc" {
  type = "generic_json"
  name = "tf-acc-%s"
  config = {
    url = "https://example.com/hook"
  }
}`, suffix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("realuptime_notification_channel.acc", "id"),
					resource.TestCheckResourceAttr("realuptime_notification_channel.acc", "enabled", "true"),
					resource.TestCheckResourceAttrSet("realuptime_notification_channel.acc", "config_summary"),
				),
			},
			{
				Config: fmt.Sprintf(`
resource "realuptime_notification_channel" "acc" {
  type    = "generic_json"
  name    = "tf-acc-%s-2"
  enabled = false
  config = {
    url = "https://example.com/hook"
  }
}`, suffix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("realuptime_notification_channel.acc", "enabled", "false"),
				),
			},
		},
	})
}
