data "cloudfly_instance_price" "example" {
  flavor_type = "Standard"
  ram         = 1
  disk        = 20
  vcpus       = 1
  region      = "HN-Cloud01"
  image_name  = "CentOS-7.9"
}
