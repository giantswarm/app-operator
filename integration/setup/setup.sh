#!/bin/bash

curl -L https://github.com/giantswarm/apptestctl/releases/download/v0.12.0/apptestctl-v0.12.0-linux-amd64.tar.gz > ./apptestctl.tar.gz
tar xzvf apptestctl.tar.gz
chmod u+x apptestctl-v0.12.0-linux-amd64/apptestctl
sudo mv apptestctl-v0.12.0-linux-amd64/apptestctl /usr/local/bin

apptestctl bootstrap --kubeconfig="$(kind get kubeconfig)" --install-operators=false --wait=false
