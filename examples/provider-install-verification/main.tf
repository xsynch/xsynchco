terraform {
  required_providers {
    xsynchco = {
      source = "jstrickland.xyz/cloud/xsynchco"
    }
  }
}

provider "xsynchco" {
  cloud_provider = "aws"
}

# data "xsynchco_bucket_aws" "example" {}

