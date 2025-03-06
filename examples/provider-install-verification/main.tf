terraform {
  required_providers {
    xsynchco = {
      source = "jstrickland.xyz/cloud/xsynchco"
    }
  }
}

provider "xsynchco" {
  cloud_provider = "aws"
  region = "us-east-1"
}

data "xsynchco_aws" "example" {}

output "all_buckets" {
  value = data.xsynchco_aws.example 
}

