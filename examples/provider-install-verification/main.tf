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

resource "xsynchco_s3_storage" "example" {

 buckets = [{

   name = "jds-test-bucket-2398756"

   tags = "mybucket"

 }]

}


# data "xsynchco_aws" "example" {}

# output "all_buckets" {
#   value = data.xsynchco_aws.example 
# }

