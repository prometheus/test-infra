variable "region" {}

provider "google" {
  region = "${var.region}"
}

variable "cluster_name" {
  default = "prombench"
}
variable "project" {}

variable "zone" {
  default = "us-east1-b"
}
variable "kubernetes_version" {
  default = "1.10.4-gke.2"
}

resource "google_container_cluster" "primary" {
  name = "${var.cluster_name}"
  project = "${var.project}"
  zone = "${var.zone}"
  initial_node_count = 1

  min_master_version = "${var.kubernetes_version}"
  node_version = "${var.kubernetes_version}"

  node_config {
    machine_type = "n1-standard-2"
    labels {
      isolation = "prow"
    }
  }
}

output "cluster_name" {
  value = "${google_container_cluster.primary.name}"
}

output "primary_zone" {
  value = "${google_container_cluster.primary.zone}"
}

output "node_version" {
  value = "${google_container_cluster.primary.node_version}"
}