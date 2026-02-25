#!/usr/bin/env bash

echo 'apiVersion: v1' > prombench/manifests/cluster-infra/grafana_dashboard_dashboards_noparse.yaml
echo 'kind: ConfigMap' >> prombench/manifests/cluster-infra/grafana_dashboard_dashboards_noparse.yaml
echo 'metadata:' >> prombench/manifests/cluster-infra/grafana_dashboard_dashboards_noparse.yaml
echo '  name: grafana-dashboards' >> prombench/manifests/cluster-infra/grafana_dashboard_dashboards_noparse.yaml
echo '  annotations:' >> prombench/manifests/cluster-infra/grafana_dashboard_dashboards_noparse.yaml
echo '    kubectl.kubernetes.io/last-applied-configuration: "-"' >> prombench/manifests/cluster-infra/grafana_dashboard_dashboards_noparse.yaml
echo 'data:' >> prombench/manifests/cluster-infra/grafana_dashboard_dashboards_noparse.yaml

# Loop over files in prombench/manifests/cluster-infra/dashboards.
for file in $(ls prombench/manifests/cluster-infra/dashboards); do
    # Read the file content.
    content=$(cat prombench/manifests/cluster-infra/dashboards/$file)
    echo "  $file: |" >> prombench/manifests/cluster-infra/grafana_dashboard_dashboards_noparse.yaml
    echo "$content" | sed 's/^/    /' >> prombench/manifests/cluster-infra/grafana_dashboard_dashboards_noparse.yaml
done