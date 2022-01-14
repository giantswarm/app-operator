#!/bin/bash

curl -s "https://raw.githubusercontent.com/giantswarm/apiextensions-application/master/config/crd/application.giantswarm.io_charts.yaml" > "../pkg/crd/charts.yaml"
