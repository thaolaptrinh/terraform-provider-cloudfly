data "cloudfly_backup_schedules" "example" {
  instance_id = cloudfly_instance.example.id
}
