#!/usr/bin/env python

import json
import os
import time
import sys
import requests
import yaml

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
    Querier launches groups of queries against a set of Prometheus services.
    """
    def __init__(self, cfg, url, req_kwargs):
        pass

    def run(self):
        pass



def main():
    if len(sys.argv) != 2 or sys.argv[1] not in ["scaler", "querier"]:
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

    config_map = requests.get(configmap_path(url, 'prometheus-load-generator'), **req_kwargs).json()
    config = yaml.load(config_map["data"]["config.json"])

    print("loaded configuration")

    if sys.argv[1] == "scaler":
        Scaler(config["scaler"], url, req_kwargs).run()
    else:
        Querier(config["querier"], url, req_kwargs).run()


if __name__ == "__main__":
    main()
