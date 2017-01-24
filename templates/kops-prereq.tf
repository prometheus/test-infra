variable "dns_domain" {}

variable "cluster_name" {}

data "aws_route53_zone" "monitoring_zone" {
  name = "${var.dns_domain}"
}

resource "aws_route53_zone" "cluster_zone" {
  name = "${var.cluster_name}.${var.dns_domain}"
}

resource "aws_route53_record" "cluster_zone_record" {
  name    = "${var.cluster_name}.${var.dns_domain}"
  zone_id = "${data.aws_route53_zone.monitoring_zone.zone_id}"
  type    = "NS"
  ttl     = "300"
  records = ["${aws_route53_zone.cluster_zone.name_servers}"]
}

resource "aws_s3_bucket" "kops-state" {
  bucket = "kops-${sha1("${var.cluster_name}-${var.dns_domain}")}"
}
