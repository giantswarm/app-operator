package controllercontext

import (
	"context"

	"github.com/giantswarm/apiextensions/v3/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/helmclient/v4/pkg/helmclient"
	"github.com/giantswarm/k8sclient/v5/pkg/k8sclient"
	"github.com/giantswarm/microerror"
)

type contextKey string

const controllerKey contextKey = "controller"

type Context struct {
	Catalog v1alpha1.Catalog
	Clients Clients
	Status  Status
}

type Clients struct {
	K8s  k8sclient.Interface
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
