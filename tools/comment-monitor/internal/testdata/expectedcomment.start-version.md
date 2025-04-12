⏱️ Welcome to Prometheus Benchmarking Tool. ⏱️

**Compared versions:** [**`PR-15487`**](http://prombench.example.com/15487/prometheus-pr) and [**`v3.0.0`**](http://prombench.example.com/15487/prometheus-release)

**Custom benchmark version:** [**`branch1` branch**](https://github.com/prometheus/test-infra/tree/branch1/prombench/manifests/prombench)

After the successful deployment ([check status here](https://github.com/prometheus/prometheus/actions/workflows/prombench.yml)), the benchmarking results can be viewed at:

- [Prometheus Meta](http://prombench.example.com/prometheus-meta/graph?g0.expr={namespace%3D"prombench-15487"}&g0.tab=1)
- [Prombench Dashboard](http://prombench.example.com/grafana/d/7gmLoNDmz/prombench?orgId=1&var-pr-number=15487)
- [Grafana Explorer, Loki logs](http://prombench.example.com/grafana/explore?orgId=1&left=["now-6h","now","loki-meta",{},{"mode":"Logs"},{"ui":[true,true,true,"none"]}])
- [Parca profiles (e.g. in-use memory)](http://prombench.example.com/profiles?expression_a=memory%3Ainuse_space%3Abytes%3Aspace%3Abytes%7Bpr_number%3D%2215487%22%7D&time_selection_a=relative:minute|15)

**Available Commands:**
* To restart benchmark: `/prombench restart v3.0.0 --bench.version=branch1`
* To stop benchmark: `/prombench cancel`
* To print help: `/prombench help`
