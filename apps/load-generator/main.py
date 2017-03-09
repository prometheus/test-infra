#!/usr/bin/env python

import json
import os
import time
import sys
import requests
import yaml
import threading

from prometheus_client import start_http_server, Histogram, Counter

ca_cert_path = "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt"
kube_token_path = "/var/run/secrets/kubernetes.io/serviceaccount/token"

def deployment_path(url, depl):
    return '%s/apis/extensions/v1beta1/namespaces/default/deployments/%s' % (url, depl)

def configmap_path(url, cm):
    return '%s/api/v1/namespaces/default/configmaps/%s' % (url, cm)

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
            time.sleep(self.interval)

            print("scaling deployment %s" % self.deployment)

            self.scale(self.low)
            time.sleep(30)
            self.scale(self.high)

    def scale(self, n):
        p = deployment_path(self.url, self.deployment)

        resp = requests.get(p, **self.req_kwargs).json()
        resp["spec"]["replicas"] = n

        requests.put(p, data=json.dumps(resp), **self.req_kwargs)


class Querier(object):
    """
    Querier launches groups of queries against a Prometheus service.
    """
    def __init__(self, i, t, qg, hist):
        self.url = "http://prometheus-test-%s.default:9090/api/v1/query?" % t
        self.interval = qg["intervalSeconds"]
        self.queries = qg["queries"]
        self.i = i
        self.t = t

        self.query_time = hist

    def run(self):
        print("run querier %s %s" % (self.t, self.i))

        while True:
            start = time.time()

            for q in self.queries:
                self.query(q)

            wait = self.interval - (time.time() - start)
            if wait > 0:
                time.sleep(wait)

    def query(self, q):
        start = time.time()
        resp = requests.get(self.url, params={"query": q})
        
        dur = time.time() - start
        print("query %s %s, status=%s, size=%d, dur=%d" %(self.t, q, resp.status_code, len(resp.text), dur))

        self.query_time.labels(self.t, str(self.i), q).observe(dur)


def main():
    if len(sys.argv) < 2 or sys.argv[1] not in ["scaler", "querier"]:
        print("unexpected arguments")
        print("usage: <load_generator> <scaler|querier>")
        exit(2)

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

    hist = Histogram("loadgen_query_duration_seconds", "Query duration", 
            ["prometheus", "group", "query"],
            buckets=(0.01, 0.05, 0.1, 0.3, 0.7, 1.5, 3, 6, 12, 18, 28))

    for t in config["querier"]["targets"]:
        i = 0
        for g in config["querier"]["queryGroups"]:
            p = threading.Thread(target=Querier(i, t["name"], g, hist).run)
            p.start()
            i += 1

    start_http_server(8080)
    print("started HTTP server on 8080")

    while True:
        time.sleep(100)

if __name__ == "__main__":
    main()
