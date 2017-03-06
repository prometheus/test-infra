#!/usr/bin/env python

import requests
import json
import yaml
import schedule
import time
import os

config = None
kubeToken = None
prevScalingUp = True
kubernetesServiceHostUrl = os.environ['KUBERNETES_SERVICE_HOST']
kubernetesPort443TCPPort = os.environ['KUBERNETES_PORT_443_TCP_PORT']
url = 'https://' + kubernetesServiceHostUrl + ':' + kubernetesPort443TCPPort
urlDeployments = url + '/apis/extensions/v1beta1/namespaces/default/deployments/'
urlConfigMaps = url + '/api/v1/namespaces/default/configmaps/'
verify = "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt"

with open("/var/run/secrets/kubernetes.io/serviceaccount/token", 'r') as f:
    kubeToken = f.read()
headers = {'Content-type': 'application/json', 'Authorization': 'Bearer ' + kubeToken}

def getConfig():
    print("Updating k8s-coach-config")
    global headers
    global verify
    global config
    configMap = requests.get((urlConfigMaps + 'k8s-simulator-config'), headers=headers, verify=verify).json()
    config = yaml.load(configMap["data"]["k8s-simulator-config.yml"])
    print("Updated k8s-coach-config")

def scaleDeployment(amountInstances):
    global headers
    global verify
    fakeWebserverDeploy = requests.get((urlDeployments + config["name"]), headers=headers, verify=verify).json()
    print("Scaling " + config["name"] + " to: " + str(amountInstances))
    fakeWebserverDeploy["spec"]["replicas"] = amountInstances
    requests.put(urlDeployments + "fake-webserver/", data=json.dumps(fakeWebserverDeploy), headers=headers, verify=verify)

def alternateUpDownScaling():
    global prevScalingUp
    if prevScalingUp:
        scaleDeployment(config["low"])
    else:
        scaleDeployment(config["high"])
    prevScalingUp = not prevScalingUp

getConfig()
schedule.every(30).seconds.do(getConfig)
schedule.every(config["interval"]).seconds.do(alternateUpDownScaling)

while True:
    schedule.run_pending()
    time.sleep(1)
