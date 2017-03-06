#!/usr/bin/env python

import json
import os
import requests
import schedule
import time
import yaml

kubeToken = None
verify = "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt"
kubernetesServiceHostUrl = os.environ['KUBERNETES_SERVICE_HOST']
kubernetesPort443TCPPort = os.environ['KUBERNETES_PORT_443_TCP_PORT']
url = 'https://' + kubernetesServiceHostUrl + ':' + kubernetesPort443TCPPort
urlDeployments = url + '/apis/extensions/v1beta1/namespaces/default/deployments/'
urlConfigMaps = url + '/api/v1/namespaces/default/configmaps/'
prevScalingUp = True
config = None

with open("/var/run/secrets/kubernetes.io/serviceaccount/token", 'r') as f:
    kubeToken = f.read()
headers = {'Content-type': 'application/json', 'Authorization': 'Bearer ' + kubeToken}

def loadConfig():
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

loadConfig()
schedule.every(config["interval"]).minutes.do(alternateUpDownScaling)

while True:
    schedule.run_pending()
    time.sleep(1)
