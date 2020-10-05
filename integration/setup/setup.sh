#!/bin/bash

go get github.com/giantswarm/apptestctl@operator-flag
apptestctl bootstrap --install-operators=false --kubeconfig="$(kind get kubeconfig)"
