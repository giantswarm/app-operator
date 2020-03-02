#!/usr/bin/env bash

kubectl label namespace default foo=bar
kubectl describe namespace default