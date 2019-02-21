package service

import (
	"github.com/giantswarm/app-operator/flag/service/kubernetes"
)

// AppCatalog is a data structure to hold AppCatalog specific command line configuration flags.
type AppCatalog struct {
	IndexNamespace string
}

// Service is an intermediate data structure for command line configuration flags.
type Service struct {
	AppCatalog AppCatalog
	Kubernetes kubernetes.Kubernetes
}
