#!/usr/bin/env bash

echo "Dummy commands for testing"
kubectl label namespace default bar=foo
kubectl describe namespace default
