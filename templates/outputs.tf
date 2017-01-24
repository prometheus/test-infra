output "kops_state_bucket" {
  value = "s3://${aws_s3_bucket.kops-state.id}"
}
