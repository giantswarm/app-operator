package service

import (
	"github.com/giantswarm/app-operator/flag/service/appcatalog"
	"github.com/giantswarm/app-operator/flag/service/chart"
	"github.com/giantswarm/app-operator/flag/service/kubernetes"
)

// Service is an intermediate data structure for command line configuration flags.
type Service struct {
	AppCatalog appcatalog.AppCatalog
	Chart      chart.Chart
	Kubernetes kubernetes.Kubernetes
}
