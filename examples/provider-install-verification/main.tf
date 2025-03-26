terraform {
  required_providers {
    xsynchco = {
      source = "jstrickland.xyz/cloud/xsynchco"
    }
  }
}

provider "xsynchco" {
  cloud_provider = "azure"
  region = "eastus"
}

# resource "xsynchco_s3_storage" "example" {

#  buckets = [{

#    name = "jds-test-bucket-2398756"

#    tags = "mybucket"

#  }]

# }

resource "xsynchco_az_storage" "example" {
  resource_group_name ="jds123abc"
  subscriptionid = "266c70b4-e30e-4d65-bac0-6c57c47a567b"

 storage_accounts = [{
  

   name = "jdstest2398756"

   tags = "mybucket"

 }]

}



# data "xsynchco_aws" "example" {}

# output "all_buckets" {
#   value = data.xsynchco_aws.example 
# }

