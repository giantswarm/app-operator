package controllercontext

import (
	"context"
	"k8s.io/client-go/rest"

	"github.com/giantswarm/apiextensions/pkg/clientset/versioned"
	"github.com/giantswarm/microerror"
	"k8s.io/client-go/kubernetes"
)

type contextKey string

const controllerKey contextKey = "controller"

type Context struct {
	G8sClient  versioned.Interface
	K8sClient  kubernetes.Interface
	RESTConfig *rest.Config
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
