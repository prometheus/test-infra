workflow "Start benchmark" {
  on = "issue_comment"
  resolves = ["start_benchmark"]
}

workflow "Cancel benchmark" {
  on = "issue_comment"
  resolves = ["cancel_benchmark"]
}

action "start_benchmark_validate" {
  uses = "docker://niki1905/cmon2:latest"
  args = ["(?mi)^/benchmark\\s*(master|[0-9]+\\.[0-9]+\\.[0-9]+\\S*)?\\s*$"]
  secrets = ["GITHUB_TOKEN"]
  env = {
	  COMMENT_TEMPLATE="Starting benchmark"
	}
}

action "cancel_benchmark_validate" {
  uses = "docker://niki1905/cmon2:latest"
  args = ["(?mi)^/benchmark\\s+cancel\\s*$"]
  secrets = ["GITHUB_TOKEN"]
  env = {
	  COMMENT_TEMPLATE="Cancelling benchmark"
	}
}

action "start_benchmark" {
  needs = ["start_benchmark_validate"]
  uses = "docker://prombench/prombench:2.0.2"
  args = [
    "export RELEASE=$(cat /github/home/ARG_0) && [ -z $RELEASE ] && export RELEASE=master",
    "export PR_NUMBER=$(cat /github/home/ARG_1);",
    "make",
    "deploy"
  ]
  secrets = ["AUTH_FILE"]
  env = {
    PROJECT_ID="prombench-example",
    CLUSTER_NAME="prombench",
    ZONE="us-central1-a",
    DOMAIN_NAME="http://prombench.prometheus.io",
    PROMBENCH_REPO="https://github.com/prometheus/prombench.git"
	}
}

action "cancel_benchmark" {
  needs = ["cancel_benchmark_validate"]
  uses = "docker://prombench/prombench:2.0.2"
  args = ["make", "clean"]
  secrets = ["AUTH_FILE"]
  env = {
    PROJECT_ID="prombench-example",
    CLUSTER_NAME="prombench",
    ZONE="us-central1-a",
    DOMAIN_NAME="http://prombench.prometheus.io"
    PROMBENCH_REPO="https://github.com/prometheus/prombench.git"
	}
}
