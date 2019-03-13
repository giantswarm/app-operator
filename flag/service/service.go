package service

import (
	"github.com/giantswarm/operatorkit/flag/service/kubernetes"

	"github.com/giantswarm/app-operator/flag/service/appcatalog"
	"github.com/giantswarm/app-operator/flag/service/chart"
)

// Service is an intermediate data structure for command line configuration flags.
type Service struct {
	AppCatalog appcatalog.AppCatalog
	Chart      chart.Chart
	Kubernetes kubernetes.Kubernetes
}
