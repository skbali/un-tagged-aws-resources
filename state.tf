terraform {
  backend "s3" {
    profile = "default"
    region  = "us-east-1"
    key     = "un-tagged-resources/terraform.tfstate"
    bucket  = "sbali-tfe-state-xyz"
  }
}