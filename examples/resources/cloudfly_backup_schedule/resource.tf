resource "cloudfly_backup_schedule" "example" {
  instance_id = cloudfly_instance.example.id
}

resource "cloudfly_backup_schedule" "daily" {
  instance_id = cloudfly_instance.example.id
  backup_type = "daily"
}
