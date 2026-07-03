resource "cloudfly_instance" "example" {
  name        = "example-instance"
  region      = "HN-Cloud01"
  flavor_type = "Standard"
  image_name  = "CentOS-7.9"
  ram         = 1
  vcpus       = 1
  disk        = 20
}
