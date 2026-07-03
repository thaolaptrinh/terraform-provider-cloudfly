data "cloudfly_instance_metrics" "cpu_1h" {
  instance_id = cloudfly_instance.example.id
  metric_type = "vcpu"
  start_time  = "1h"
}

data "cloudfly_instance_metrics" "memory_24h" {
  instance_id = cloudfly_instance.example.id
  metric_type = "memory"
  start_time  = "1d"
}

data "cloudfly_instance_metrics" "disk_7d" {
  instance_id = cloudfly_instance.example.id
  metric_type = "disk"
  start_time  = "7d"
}
