variable "tags" {
  type = map(string)
  default = {
    "CostCenter" = "infra"
  }
}

variable "region" {
  type    = string
  default = "us-east-1"
}

