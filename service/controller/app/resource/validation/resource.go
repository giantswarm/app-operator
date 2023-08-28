package validation

import (
	"github.com/giantswarm/app/v7/pkg/validation"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	Name = "validation"
)

// Config represents the configuration used to create a new chartstatus resource.
type Config struct {
	CtrlClient client.Client
	K8sClient  kubernetes.Interface
	Logger     micrologger.Logger

	ProjectName string
	Provider    string
}

// Resource implements the chartstatus resource.
type Resource struct {
	appValidator *validation.Validator
	ctrlClient   client.Client
	logger       micrologger.Logger
}

func New(config Config) (*Resource, error) {
	if config.K8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.K8sClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	if config.ProjectName == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.ProjectName must not be empty", config)
	}
	if config.Provider == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.Provider must not be empty", config)
	}

	var err error

	var appValidator *validation.Validator
	{
		c := validation.Config{
			G8sClient: config.CtrlClient,
			K8sClient: config.K8sClient,
			Logger:    config.Logger,

			ProjectName: config.ProjectName,
			Provider:    config.Provider,
		}
		appValidator, err = validation.NewValidator(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	r := &Resource{
		// Dependencies.
		appValidator: appValidator,
		ctrlClient:   config.CtrlClient,
		logger:       config.Logger,
	}

	return r, nil
}

func (r Resource) Name() string {
	return Name
}
