#!/usr/bin/env python

import json
import os
import time
import sys
import requests
import yaml
import threading
from datetime import timedelta

from prometheus_client import start_http_server, Histogram, Counter

ca_cert_path = "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt"
kube_token_path = "/var/run/secrets/kubernetes.io/serviceaccount/token"
namespace = ""
max_404_errors = 30

def deployment_path(url, depl):
    return '%s/apis/apps/v1/namespaces/%s/deployments/%s' % (url, namespace, depl)

def configmap_path(url, cm):
    return '%s/api/v1/namespaces/%s/configmaps/%s' % (url, namespace, cm)

class Scaler(object):
    """
    Scaler periodically scales a deployment down to a minimum number of
    replicas and back up again.
    """
    def __init__(self, cfg, url, req_kwargs):
        self.interval = int(cfg["intervalMinutes"]) * 60
        self.deployment = cfg["name"]
        self.low = cfg["low"]
        self.high = cfg["high"]
        self.url = url
        self.req_kwargs = req_kwargs

    def run(self):
        while True:
            print("scaling deployment %s to %d" % (self.deployment, self.low))
            self.scale(self.low)
            time.sleep(self.interval)

            print("scaling deployment %s to %d" % (self.deployment, self.high))
            self.scale(self.high)
            time.sleep(self.interval)

    def scale(self, n):
        p = deployment_path(self.url, self.deployment)

        resp = requests.get(p, **self.req_kwargs).json()
        resp["spec"]["replicas"] = n

        requests.put(p, data=json.dumps(resp), **self.req_kwargs)


class Querier(object):
    """
    Querier launches groups of queries against a Prometheus service.
    """

    query_duration = Histogram("loadgen_query_duration_seconds", "Query duration",
        ["prometheus", "group", "expr", "type"],
        buckets=(0.05, 0.1, 0.3, 0.7, 1.5, 2.5, 4, 6, 8, 10, 13, 16, 20, 24, 29, 36, 42, 50, 60))

    query_count = Counter('loadgen_queries_total', 'Total amount of queries',
        ["prometheus", "group", "expr", "type"],
    )
    query_fail_count = Counter('loadgen_failed_queries_total', 'Amount of failed queries',
        ["prometheus", "group", "expr", "type"],
    )

    def __init__(self, groupID, target, pr_number, qg):
        self.target = target
        self.name = qg["name"]
        self.groupID = groupID
        self.numberOfErrors = 0

        self.interval = duration_seconds(qg["interval"])
        self.queries = qg["queries"]
        self.type = qg.get("type", "instant")
        self.start = duration_seconds(qg.get("start", "0h"))
        self.end = duration_seconds(qg.get("end", "0h"))
        self.step = qg.get("step", "15s")

        if self.type == "instant":
            self.url = "http://prombench.prometheus.io/%s/prometheus-%s/api/v1/query" % (pr_number, target)
        else:
            self.url = "http://prombench.prometheus.io/%s/prometheus-%s/api/v1/query_range" % (pr_number, target)

    def run(self):
        print("run querier %s %s for %s" % (self.target, self.name, self.url))
        print("Waiting for 20 seconds to allow prometheus server (%s) to be properly set-up" % (self.url))
        time.sleep(20)

        while True:
            start = time.time()

            for q in self.queries:
                self.query(q["expr"])

            wait = self.interval - (time.time() - start)
            time.sleep(max(wait, 0))

    def query(self, expr):
        try:
            Querier.query_count.labels(self.target, self.name, expr, self.type).inc()
            start = time.time()

            params = {"query": expr}
            if self.type == "range":
                params["start"] = start - self.start
                params["end"] = start - self.end
                params["step"] = self.step

            resp = requests.get(self.url, params)
            dur = time.time() - start

            if resp.status_code == 404:
                print("WARNING :: GroupId#%d : Querier returned 404 for prometheus instance %s." % (self.groupID, self.url))
                self.numberOfErrors += 1
                if self.numberOfErrors == max_404_errors:
                    print("ERROR :: GroupId#%d : Querier returned 404 for prometheus instance %s %d times." % (self.groupID, self.url, max_404_errors))
                    os._exit(1)
            elif resp.status_code != 200:
                print("WARNING :: GroupId#%d : Querier returned %d for prometheus instance %s." % (self.groupID, resp.status_code, self.url))
            else:
                print("GroupId#%d : query %s %s, status=%s, size=%d, dur=%.3f" % (self.groupID, self.target, expr, resp.status_code, len(resp.text), dur))
                Querier.query_duration.labels(self.target, self.name, expr, self.type).observe(dur)

        except IOError as e:
            Querier.query_fail_count.labels(self.target, self.name, expr, self.type).inc()
            print("WARNING :: GroupId#%d : Could not query prometheus instance %s. \n %s" % (self.groupID, self.url, e))

        except Exception as e:
            Querier.query_fail_count.labels(self.target, self.name, expr, self.type).inc()
            print("WARNING :: GroupId#%d : Could not query prometheus instance %s. \n %s" % (self.groupID, self.url, e))

def duration_seconds(s):
    num = int(s[:-1])

    if s.endswith('s'):
        return timedelta(seconds=num).total_seconds()
    elif s.endswith('m'):
        return timedelta(minutes=num).total_seconds()
    elif s.endswith('h'):
        return timedelta(hours=num).total_seconds()

    raise "unknown duration %s" % s

def main():
    if len(sys.argv) < 4 or sys.argv[1] not in ["scaler", "querier"]:
        print("unexpected arguments")
        print("usage: <load_generator> <scaler|querier> <namespace> <pr_number>")
        exit(2)

    global namespace
    namespace = sys.argv[2]
    pr_number = sys.argv[3]

    host = os.environ.get('KUBERNETES_SERVICE_HOST')
    port = os.environ.get('KUBERNETES_PORT_443_TCP_PORT')
    url = 'https://%s:%s' % (host, port)

    kube_token = open(kube_token_path, 'r').read()

    req_kwargs = {
        'headers': {'Content-type': 'application/json', 'Authorization': 'Bearer ' + kube_token},
        'verify': ca_cert_path,
    }

    config = yaml.load(open("/etc/loadgen/config.yaml", 'r').read())

    print("loaded configuration")

    if sys.argv[1] == "scaler":
        Scaler(config["scaler"], url, req_kwargs).run()
        return

    if sys.argv[1] == "querier":
        for i,g in enumerate(config["querier"]["groups"]):
            p = threading.Thread(target=Querier(i, "pr", pr_number, g).run)
            p.start()

        for i,g in enumerate(config["querier"]["groups"]):
            p = threading.Thread(target=Querier(i, "release", pr_number, g).run)
            p.start()

    start_http_server(8080)
    print("started HTTP server on 8080")

    while True:
        time.sleep(100)

if __name__ == "__main__":
    main()