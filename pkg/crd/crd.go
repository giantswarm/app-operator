package crd

import (
	_ "embed"
)

//go:embed charts.yaml
var charts string

func ChartCRD() string {
	return charts
}
