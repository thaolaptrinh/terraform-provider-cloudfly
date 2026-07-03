resource "cloudfly_snapshot" "example" {
  instance_id = cloudfly_instance.example.id
  name        = "before-upgrade"
  description = "Snapshot before system upgrade"
}

resource "cloudfly_snapshot" "clean" {
  instance_id = cloudfly_instance.example.id
  name        = "post-install"
}
