package service

import (
	"github.com/giantswarm/operatorkit/v4/pkg/flag/service/kubernetes"

	"github.com/giantswarm/app-operator/v3/flag/service/app"
	"github.com/giantswarm/app-operator/v3/flag/service/chart"
	"github.com/giantswarm/app-operator/v3/flag/service/helm"
	"github.com/giantswarm/app-operator/v3/flag/service/image"
)

// Service is an intermediate data structure for command line configuration flags.
type Service struct {
	App        app.App
	Chart      chart.Chart
	Helm       helm.Helm
	Image      image.Image
	Kubernetes kubernetes.Kubernetes
}
