terraform {
  required_providers {
    realuptime = {
      source  = "RealUptimeHQ/realuptime"
      version = "~> 0.1"
    }
  }
}

# The key comes from REALUPTIME_API_KEY; set api_key here only from a secret
# store, never a literal. base_url is optional (REALUPTIME_API_URL also works).
provider "realuptime" {}
