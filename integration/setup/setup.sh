#!/bin/bash

go get github.com/giantswarm/apptestctl@operator-flag
apptestctl bootstrap --kubeconfig="$(kind get kubeconfig)"