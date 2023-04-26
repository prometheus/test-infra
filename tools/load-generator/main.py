#!/usr/bin/env python

import os
import time
import sys
import requests
import yaml
import threading
from datetime import timedelta

from prometheus_client import start_http_server, Histogram, Counter

namespace = ""
max_404_errors = 30
domain_name = os.environ["DOMAIN_NAME"]

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
            self.url = "http://%s/%s/prometheus-%s/api/v1/query" % (domain_name, pr_number, target)
        else:
            self.url = "http://%s/%s/prometheus-%s/api/v1/query_range" % (domain_name, pr_number, target)

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
    if len(sys.argv) < 3:
        print("unexpected arguments")
        print("usage: <load_generator> <namespace> <pr_number>")
        exit(2)

    global namespace
    namespace = sys.argv[1]
    pr_number = sys.argv[2]

    config = yaml.load(open("/etc/loadgen/config.yaml", 'r', Loader=yaml.FullLoader).read())

    print("loaded configuration")

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
