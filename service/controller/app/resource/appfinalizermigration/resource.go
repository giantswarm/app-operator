package appfinalizermigration

import (
	"context"
	"strings"

	"github.com/giantswarm/app/v5/pkg/key"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// Name is the identifier of the resource.
	Name = "appfinalizermigration"
)

var (
	// legacyFinalizers are removed by this resource.
	legacyFinalizers = map[string]bool{
		"operatorkit.giantswarm.io/app":          true,
		"operatorkit.giantswarm.io/app-operator": true,
	}
)

type Config struct {
	CtrlClient client.Client
	Logger     micrologger.Logger
}

// Resource does garbage collection on the App CR finalizers.
type Resource struct {
	ctrlClient client.Client
	logger     micrologger.Logger
}

func New(config Config) (*Resource, error) {
	if config.CtrlClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.CtrlClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	r := &Resource{
		ctrlClient: config.CtrlClient,
		logger:     config.Logger,
	}

	return r, nil
}

// EnsureCreated ensures that reconciled App CR gets orphaned finalizer
// deleted.
func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	cr, err := key.ToApp(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	{
		// Refresh the CR object.
		err := r.ctrlClient.Get(ctx, client.ObjectKey{Name: cr.Name, Namespace: cr.Namespace}, &cr)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	var exists bool
	var newFinalizers []string
	for _, v := range cr.Finalizers {
		if legacyFinalizers[strings.TrimSpace(v)] {
			// drop it
			exists = true
			continue
		}

		newFinalizers = append(newFinalizers, strings.TrimSpace(v))
	}

	if exists {
		cr.Finalizers = newFinalizers
		r.logger.Debugf(ctx, "deleting legacy finalizer from app CR")

		err := r.ctrlClient.Update(ctx, &cr)
		if err != nil {
			return microerror.Mask(err)
		}

		r.logger.Debugf(ctx, "deleted legacy finalizer from app CR")
		return nil
	}

	return nil
}

// EnsureDeleted is no-op.
func (r *Resource) EnsureDeleted(ctx context.Context, obj interface{}) error {
	return nil
}

// Name returns the resource name.
func (r *Resource) Name() string {
	return Name
}
