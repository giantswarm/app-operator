#!/bin/bash

apptestctl bootstrap --kubeconfig="$(kind get kubeconfig)" --install-operators=false

PROMETHEUS_OPERATOR_VERSION="v0.56.3"

KUBE_CONFIG=$(kind get kubeconfig) kubectl apply -f "https://raw.githubusercontent.com/prometheus-operator/prometheus-operator/${PROMETHEUS_OPERATOR_VERSION}/example/prometheus-operator-crd/monitoring.coreos.com_podmonitors.yaml"

KUBE_CONFIG=$(kind get kubeconfig) kubectl apply -f "https://raw.githubusercontent.com/prometheus-operator/prometheus-operator/${PROMETHEUS_OPERATOR_VERSION}/example/prometheus-operator-crd/monitoring.coreos.com_servicemonitors.yaml"

## This is hack for the time being until a new apptestctl is released
KUBE_CONFIG=$(kind get kubeconfig) kubectl apply -f "https://raw.githubusercontent.com/giantswarm/apiextensions-application/main/config/crd/application.giantswarm.io_catalogs.yaml"
