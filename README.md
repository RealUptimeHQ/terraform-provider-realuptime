# Terraform Provider: RealUptime

Manage [RealUptime](https://realuptime.io) monitors, status pages, status page
components and notification channels as code.

**Status: live on the Terraform Registry** at
[registry.terraform.io/providers/RealUptimeHQ/realuptime](https://registry.terraform.io/providers/RealUptimeHQ/realuptime).
`terraform init` resolves the source address `RealUptimeHQ/realuptime`
directly; no `dev_overrides` block is needed. Building locally (below) is
only for contributors.

## Resources and data sources

| Name | Maps to |
|---|---|
| `realuptime_check` | `POST/GET/PATCH/DELETE /api/v1/checks` (http monitors; regions and assertions updated in place) |
| `realuptime_status_page` | `POST/GET/PATCH/DELETE /api/v1/status-pages` |
| `realuptime_status_page_component` | `/api/v1/status-pages/{id}/components` |
| `realuptime_notification_channel` | `/api/v1/notification-channels` (Discord, Teams, Opsgenie, generic JSON) |
| `data.realuptime_status_page` | `GET /api/v1/status-pages[/{id}]`, by id or slug |

Every attribute is hand-mapped from the OpenAPI 3.1 document at
`https://realuptime.io/api/v1/openapi.json`, with the API's own field names.
Full attribute reference: [`docs/`](./docs) (generated from the schemas by
`go run ./cmd/gendocs`; a test fails if they drift).

## Requirements

- Terraform 1.5+ (protocol 6)
- A RealUptime API key with **read_write** permission, on any
  plan, exported as `REALUPTIME_API_KEY`. Generate one from the dashboard's
  API & MCP access section.

## Usage

```terraform
terraform {
  required_providers {
    realuptime = {
      source  = "RealUptimeHQ/realuptime"
      version = "~> 0.1"
    }
  }
}

provider "realuptime" {} # reads REALUPTIME_API_KEY

resource "realuptime_check" "api" {
  name    = "Public API"
  url     = "https://api.example.com/health"
  regions = ["iad", "fra"]
}

data "realuptime_status_page" "main" {
  slug = "acme"
}

resource "realuptime_status_page_component" "api" {
  status_page_id = data.realuptime_status_page.main.id
  check_id       = realuptime_check.api.id
}
```

More in [`examples/`](./examples).

## Building and using locally (contributors)

```sh
go build -o terraform-provider-realuptime .
```

`~/.terraformrc`:

```hcl
provider_installation {
  dev_overrides {
    "RealUptimeHQ/realuptime" = "/absolute/path/to/packages/terraform-provider-realuptime"
  }
  direct {}
}
```

Then `terraform plan` in any configuration that declares the provider (skip
`terraform init` for dev overrides).

## Tests

```sh
go test ./...                                 # unit: client mapping (httptest), schemas, docs freshness
TF_ACC=1 REALUPTIME_API_KEY=ru_live_... \
REALUPTIME_API_URL=http://localhost:3000/api/v1 \
go test ./internal/provider -run TestAcc -v   # acceptance: real resources on that account
```

Acceptance tests need a `terraform` binary on `PATH` and create/destroy
resources named `tf-acc-<n>` on the account the key belongs to.

## Releasing (maintainers)

Releases are cut by GoReleaser on a `v*` tag (`.github/workflows/release.yml`,
`.goreleaser.yml`), signed with the GPG key registered on the Terraform
Registry.

## License

MIT.
