provider "edgecenter" {
  permanent_api_token = "251$d3361.............1b35f26d8"
}

data "edgecenter_k8s" "cluster" {
  project_id = 1
  region_id  = 1
  cluster_id = "dc3a3ea9-86ae-47ad-a8e8-79df0ce04839"
}

