package controllercontext

import (
	"context"

	"github.com/giantswarm/apiextensions/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/apiextensions/pkg/clientset/versioned"
	"github.com/giantswarm/helmclient"
	"github.com/giantswarm/microerror"
	"k8s.io/client-go/kubernetes"
)

type contextKey string

const controllerKey contextKey = "controller"

type Context struct {
	AppCatalog v1alpha1.AppCatalog
	Clients    Clients
	Status     Status
}

type Clients struct {
	G8s  versioned.Interface
	K8s  kubernetes.Interface
	Helm helmclient.Interface
}

type Status struct {
	TenantCluster TenantCluster
}

type TenantCluster struct {
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
