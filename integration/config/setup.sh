#!/usr/bin/env bash

echo "Dummy commands for testing"
kubectl label namespace default foo=bar
kubectl describe namespace default
