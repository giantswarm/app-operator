package service

import (
	"github.com/giantswarm/app-operator/v7/flag/service/app"
	"github.com/giantswarm/app-operator/v7/flag/service/appcatalog"
	"github.com/giantswarm/app-operator/v7/flag/service/chart"
	"github.com/giantswarm/app-operator/v7/flag/service/helm"
	"github.com/giantswarm/app-operator/v7/flag/service/image"
	"github.com/giantswarm/app-operator/v7/flag/service/kubernetes"
	"github.com/giantswarm/app-operator/v7/flag/service/operatorkit"
	"github.com/giantswarm/app-operator/v7/flag/service/provider"
)

// Service is an intermediate data structure for command line configuration flags.
type Service struct {
	App         app.App
	AppCatalog  appcatalog.AppCatalog
	Chart       chart.Chart
	Helm        helm.Helm
	Image       image.Image
	Kubernetes  kubernetes.Kubernetes
	Operatorkit operatorkit.Operatorkit
	Provider    provider.Provider
}
