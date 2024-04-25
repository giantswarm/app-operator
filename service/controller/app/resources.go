package app

import (
	"time"

	"github.com/giantswarm/app/v7/pkg/values"
	"github.com/giantswarm/helmclient/v4/pkg/helmclient"
	"github.com/giantswarm/k8sclient/v7/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/operatorkit/v8/pkg/resource"
	"github.com/giantswarm/operatorkit/v8/pkg/resource/crud"
	"github.com/giantswarm/operatorkit/v8/pkg/resource/wrapper/metricsresource"
	"github.com/giantswarm/operatorkit/v8/pkg/resource/wrapper/retryresource"
	"github.com/spf13/afero"

	"github.com/giantswarm/app-operator/v6/service/controller/app/resource/appfinalizermigration"
	"github.com/giantswarm/app-operator/v6/service/controller/app/resource/appnamespace"
	"github.com/giantswarm/app-operator/v6/service/controller/app/resource/catalog"
	"github.com/giantswarm/app-operator/v6/service/controller/app/resource/chart"
	"github.com/giantswarm/app-operator/v6/service/controller/app/resource/chartcrd"
	"github.com/giantswarm/app-operator/v6/service/controller/app/resource/chartoperator"
	"github.com/giantswarm/app-operator/v6/service/controller/app/resource/clients"
	"github.com/giantswarm/app-operator/v6/service/controller/app/resource/configmap"
	"github.com/giantswarm/app-operator/v6/service/controller/app/resource/helmrelease"
	"github.com/giantswarm/app-operator/v6/service/controller/app/resource/migration"
	"github.com/giantswarm/app-operator/v6/service/controller/app/resource/secret"
	"github.com/giantswarm/app-operator/v6/service/controller/app/resource/status"
	"github.com/giantswarm/app-operator/v6/service/controller/app/resource/tcnamespace"
	"github.com/giantswarm/app-operator/v6/service/controller/app/resource/validation"
	"github.com/giantswarm/app-operator/v6/service/internal/clientcache"
	"github.com/giantswarm/app-operator/v6/service/internal/indexcache"
)

type appResourcesConfig struct {
	// Dependencies.
	ClientCache *clientcache.Resource
	FileSystem  afero.Fs
	IndexCache  indexcache.Interface
	K8sClient   k8sclient.Interface
	Logger      micrologger.Logger

	// Settings.
	ChartNamespace                     string
	HelmControllerBackend              bool
	HelmControllerBackendAutoMigration bool
	HTTPClientTimeout                  time.Duration
	ImageRegistry                      string
	ProjectName                        string
	Provider                           string
	UniqueApp                          bool
	WorkloadClusterID                  string
	DependencyWaitTimeoutMinutes       int
}

func newAppResources(config appResourcesConfig) ([]resource.Interface, error) {
	var err error

	// Dependencies.
	if config.ClientCache == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.ClientCache must not be empty", config)
	}
	if config.FileSystem == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Fs must not be empty", config)
	}
	if config.IndexCache == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.IndexCache must not be empty", config)
	}
	if config.K8sClient == k8sclient.Interface(nil) {
		return nil, microerror.Maskf(invalidConfigError, "%T.K8sClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	// Settings.
	if config.ChartNamespace == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.ChartNamespace must not be empty", config)
	}
	if config.HTTPClientTimeout == 0 {
		return nil, microerror.Maskf(invalidConfigError, "%T.HTTPClientTimeout must not be empty", config)
	}
	if config.ImageRegistry == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.ImageRegistry must not be empty", config)
	}
	if config.ProjectName == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.ProjectName must not be empty", config)
	}
	if config.Provider == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.Provider must not be empty", config)
	}

	var valuesService *values.Values
	{
		c := values.Config{
			K8sClient: config.K8sClient.K8sClient(),
			Logger:    config.Logger,
		}

		valuesService, err = values.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var catalogResource resource.Interface
	{
		c := catalog.Config{
			CtrlClient: config.K8sClient.CtrlClient(),
			Logger:     config.Logger,
		}
		catalogResource, err = catalog.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var appFinalizerResource resource.Interface
	{
		c := appfinalizermigration.Config{
			CtrlClient: config.K8sClient.CtrlClient(),
			Logger:     config.Logger,
		}
		appFinalizerResource, err = appfinalizermigration.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var appNamespaceResource resource.Interface
	{
		c := appnamespace.Config{
			K8sClient: config.K8sClient.K8sClient(),
			Logger:    config.Logger,
		}
		appNamespaceResource, err = appnamespace.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var chartOperatorResource resource.Interface
	{
		c := chartoperator.Config{
			FileSystem: config.FileSystem,
			CtrlClient: config.K8sClient.CtrlClient(),
			K8sClient:  config.K8sClient.K8sClient(),
			Logger:     config.Logger,
			Values:     valuesService,

			ChartNamespace:    config.ChartNamespace,
			WorkloadClusterID: config.WorkloadClusterID,
		}
		chartOperatorResource, err = chartoperator.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var chartResource resource.Interface
	{
		c := chart.Config{
			IndexCache:    config.IndexCache,
			Logger:        config.Logger,
			CtrlClient:    config.K8sClient.CtrlClient(),
			DynamicClient: config.K8sClient.DynClient(),

			ChartNamespace:               config.ChartNamespace,
			WorkloadClusterID:            config.WorkloadClusterID,
			DependencyWaitTimeoutMinutes: config.DependencyWaitTimeoutMinutes,
		}

		ops, err := chart.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		chartResource, err = toCRUDResource(config.Logger, ops)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var chartCRDResource resource.Interface
	{
		c := chartcrd.Config{
			Logger: config.Logger,
		}

		chartCRDResource, err = chartcrd.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var helmClient helmclient.Interface
	{
		c := helmclient.Config{
			Fs:         config.FileSystem,
			K8sClient:  config.K8sClient.K8sClient(),
			Logger:     config.Logger,
			RestClient: config.K8sClient.RESTClient(),
			RestConfig: config.K8sClient.RESTConfig(),

			HTTPClientTimeout: config.HTTPClientTimeout,
		}

		helmClient, err = helmclient.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var clientsResource resource.Interface
	{
		c := clients.Config{
			ClientCache:           config.ClientCache,
			HelmClient:            helmClient,
			HelmControllerBackend: config.HelmControllerBackend,
			K8sClient:             config.K8sClient,
			Logger:                config.Logger,
		}

		clientsResource, err = clients.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var configMapResource resource.Interface
	{
		c := configmap.Config{
			Logger: config.Logger,
			Values: valuesService,
		}

		if config.HelmControllerBackend {
			c.HelmControllerBackend = true
		} else {
			c.ChartNamespace = config.ChartNamespace
		}

		ops, err := configmap.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		configMapResource, err = toCRUDResource(config.Logger, ops)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var helmReleaseResource resource.Interface
	{
		c := helmrelease.Config{
			IndexCache: config.IndexCache,
			Logger:     config.Logger,
			CtrlClient: config.K8sClient.CtrlClient(),

			WorkloadClusterID:            config.WorkloadClusterID,
			DependencyWaitTimeoutMinutes: config.DependencyWaitTimeoutMinutes,
		}

		ops, err := helmrelease.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		helmReleaseResource, err = toCRUDResource(config.Logger, ops)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var migrationResource resource.Interface
	{
		c := migration.Config{
			CtrlClient: config.K8sClient.CtrlClient(),
			Logger:     config.Logger,

			ChartNamespace:    config.ChartNamespace,
			WorkloadClusterID: config.WorkloadClusterID,
		}

		migrationResource, err = migration.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var secretResource resource.Interface
	{
		c := secret.Config{
			Logger: config.Logger,
			Values: valuesService,
		}

		if config.HelmControllerBackend {
			c.HelmControllerBackend = true
		} else {
			c.ChartNamespace = config.ChartNamespace
		}

		ops, err := secret.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		secretResource, err = toCRUDResource(config.Logger, ops)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var statusResource resource.Interface
	{
		c := status.Config{
			CtrlClient: config.K8sClient.CtrlClient(),
			Logger:     config.Logger,

			ChartNamespace:        config.ChartNamespace,
			HelmControllerBackend: config.HelmControllerBackend,
			WorkloadClusterID:     config.WorkloadClusterID,
		}

		statusResource, err = status.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var tcNamespaceResource resource.Interface
	{
		c := tcnamespace.Config{
			Logger: config.Logger,
		}

		tcNamespaceResource, err = tcnamespace.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var validationResource resource.Interface
	{
		c := validation.Config{
			CtrlClient: config.K8sClient.CtrlClient(),
			K8sClient:  config.K8sClient.K8sClient(),
			Logger:     config.Logger,

			ProjectName: config.ProjectName,
			Provider:    config.Provider,
		}

		validationResource, err = validation.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	resources := []resource.Interface{
		// validationResource checks CRs for validation errors and sets the CR status.
		validationResource,

		// appFinalizerResource check CRs for legacy finalizers and removes them.
		appFinalizerResource,

		// Following resources manage controller context information. The `clientsResource`
		// resource is being used regardless of downstream controller, that is either Helm
		// Controller or Chart Operator, because it has been previously used regardless of
		// App Operator uniqueness or not. The reason for that was consistency, and hence it
		// remains a good reason to use it regardless of the downstream controller. Changing
		// this logic now would require more code changes which is riskier, taking into
		// account the amount of so far changes, without any impactful gain.
		//
		// It is however fair to note, without Chart Operator we do nothing in a workload cluster,
		// hence these context-based clients are not needed, with one exception for the migration
		// period.
		appNamespaceResource,
		catalogResource,
		clientsResource,
	}

	if !config.HelmControllerBackend {
		// Following resources bootstrap chart-operator in workload clusters.
		resources = append(
			resources,
			[]resource.Interface{
				tcNamespaceResource,
				chartCRDResource,
				chartOperatorResource,
			}...,
		)
	}

	// Following resources process app CRs.
	resources = append(
		resources,
		[]resource.Interface{
			configMapResource,
			secretResource,
		}...,
	)

	if !config.HelmControllerBackend {
		// chartResource resource creates Chart CR.
		resources = append(resources, chartResource)
	} else {
		// append automated migration resource upon request
		if config.HelmControllerBackendAutoMigration {
			resources = append(resources, migrationResource)
		}
		// helmReleaseResource resource creates HelmRelease CR.
		resources = append(resources, helmReleaseResource)
	}

	resources = append(resources, statusResource)

	{
		c := retryresource.WrapConfig{
			Logger: config.Logger,
		}

		resources, err = retryresource.Wrap(resources, c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	{
		c := metricsresource.WrapConfig{}
		resources, err = metricsresource.Wrap(resources, c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	return resources, nil
}

func toCRUDResource(logger micrologger.Logger, ops crud.Interface) (*crud.Resource, error) {
	c := crud.ResourceConfig{
		Logger: logger,
		CRUD:   ops,
	}

	r, err := crud.NewResource(c)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return r, nil
}
