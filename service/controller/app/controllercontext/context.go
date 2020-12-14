package controllercontext

import (
	"context"

	"github.com/giantswarm/apiextensions/v3/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/helmclient/v4/pkg/helmclient"
	"github.com/giantswarm/microerror"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type contextKey string

const controllerKey contextKey = "controller"

type Context struct {
	AppCatalog v1alpha1.AppCatalog
	Clients    Clients
	Status     Status
}

type Clients struct {
	Ctrl client.Client
	Helm helmclient.Interface
}

type Status struct {
	ChartStatus   ChartStatus
	ClusterStatus ClusterStatus
}

type ChartStatus struct {
	Reason string
	Status string
}

type ClusterStatus struct {
	IsDeleting    bool
	IsUnavailable bool
}

func NewContext(ctx context.Context, c Context) context.Context {
	return context.WithValue(ctx, controllerKey, &c)
}

func FromContext(ctx context.Context) (*Context, error) {
	c, ok := ctx.Value(controllerKey).(*Context)
	if !ok {
		return nil, microerror.Maskf(notFoundError, "context key %q of type %T", controllerKey, controllerKey)
	}

	return c, nil
}
