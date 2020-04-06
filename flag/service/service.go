package service

import (
	"github.com/giantswarm/operatorkit/flag/service/kubernetes"

	"github.com/giantswarm/app-operator/flag/service/chart"
	"github.com/giantswarm/app-operator/flag/service/helm"
	"github.com/giantswarm/app-operator/flag/service/image"
)

// Service is an intermediate data structure for command line configuration flags.
type Service struct {
	Chart      chart.Chart
	Helm       helm.Helm
	Image      image.Image
	Kubernetes kubernetes.Kubernetes
}
