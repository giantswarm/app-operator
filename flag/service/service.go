package service

import (
	"github.com/giantswarm/app-operator/flag/service/kubernetes"
)

type AppCatalog struct {
	IndexNamespace string
}

// Service is an intermediate data structure for command line configuration flags.
type Service struct {
	Kubernetes kubernetes.Kubernetes
	AppCatalog AppCatalog
}
