package migration

import (
	"context"

	"github.com/giantswarm/apiextensions-application/api/v1alpha1"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/app-operator/v6/service/controller/app/controllercontext"
)

const (
	ChartOperatorPaused = "chart-operator.giantswarm.io/paused"

	chartCRDName = "charts.application.giantswarm.io"

	Name = "migration"
)

type Config struct {
	CtrlClient client.Client
	Logger     micrologger.Logger

	ChartNamespace    string
	WorkloadClusterID string
}

type Resource struct {
	ctrlClient client.Client
	logger     micrologger.Logger

	chartNamespace    string
	workloadClusterID string
}

type values struct {
	kind      string
	name      string
	namespace string
}

func New(config Config) (*Resource, error) {
	if config.CtrlClient == client.Client(nil) {
		return nil, microerror.Maskf(invalidConfigError, "%T.CtrlClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}
	if config.ChartNamespace == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.ChartNamespace must not be empty", config)
	}

	r := &Resource{
		// Dependencies.
		ctrlClient: config.CtrlClient,
		logger:     config.Logger,

		chartNamespace:    config.ChartNamespace,
		workloadClusterID: config.WorkloadClusterID,
	}

	return r, nil
}

func (r Resource) Name() string {
	return Name
}

func (r *Resource) removeFinalizer(ctx context.Context, chart *v1alpha1.Chart) error {
	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	if len(chart.Finalizers) == 0 {
		// Return early as nothing to do.
		return nil
	}

	r.logger.Debugf(ctx, "deleting finalizers on Chart CR %#q in namespace %#q", chart.Name, chart.Namespace)

	modifiedChart := chart.DeepCopy()
	modifiedChart.Finalizers = []string{}

	err = cc.MigrationClients.K8s.CtrlClient().Patch(ctx, modifiedChart, client.MergeFrom(chart))
	if err != nil {
		return microerror.Mask(err)
	}

	r.logger.Debugf(ctx, "deleted finalizers on Chart CR %#q in namespace %#q", chart.Name, chart.Namespace)

	return nil
}
