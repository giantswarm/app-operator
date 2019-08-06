package chartoperator

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/giantswarm/app-operator/service/controller/app/v1/key"

	"github.com/giantswarm/apiextensions/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/helmclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/operatorkit/controller/context/reconciliationcanceledcontext"
	"github.com/spf13/afero"
	"gopkg.in/yaml.v2"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/helm/pkg/helm"
)

const (
	Name = "chartoperatorv1"

	chartOperatorNamespace = "giantswarm"
	chartOperatorRelease   = "chart-operator"
	chartOperatorVersion   = "0.9.0"
)

// Config represents the configuration used to create a new clients resource.
type Config struct {
	// Dependencies.
	Fs        afero.Fs
	K8sClient kubernetes.Interface
	Logger    micrologger.Logger

	// Settings.
	RegistryDomain string
}

type Resource struct {
	// Dependencies.
	fs        afero.Fs
	k8sClient kubernetes.Interface
	logger    micrologger.Logger

	// Settings.
	registryDomain string
}

// New creates a new configured chartoperator resource.
func New(config Config) (*Resource, error) {
	if config.Fs == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Fs must not be empty", config)
	}
	if config.K8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.K8sClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	if config.RegistryDomain == "" {
		config.RegistryDomain = "quay.io"
	}

	r := &Resource{
		// Dependencies.
		fs:        config.Fs,
		k8sClient: config.K8sClient,
		logger:    config.Logger,

		// Settings.
		registryDomain: config.RegistryDomain,
	}

	return r, nil
}

func (r Resource) Name() string {
	return Name
}

func (r Resource) installChartOperator(ctx context.Context, cr v1alpha1.App, helmClient helmclient.Interface) error {
	var err error

	var tarballPath string
	{
		tarballURL := fmt.Sprintf("https://giantswarm.github.io/giantswarm-catalog/chart-operator-%s.tgz", chartOperatorVersion)
		tarballPath, err = helmClient.PullChartTarball(ctx, tarballURL)
		if err != nil {
			return microerror.Mask(err)
		}

		defer func() {
			err := r.fs.Remove(tarballPath)
			if err != nil {
				r.logger.LogCtx(ctx, "level", "error", "message", fmt.Sprintf("deletion of %#q failed", tarballPath), "stack", fmt.Sprintf("%#v", err))
			}
		}()
	}

	var chartOperatorValue []byte
	{
		name := key.ClusterValuesConfigMapName(cr)
		cm, err := r.k8sClient.CoreV1().ConfigMaps(cr.Namespace).Get(name, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			// Stop reconciliation since we can't proceed without chart-operator
			reconciliationcanceledcontext.SetCanceled(ctx)
			return nil
		} else if err != nil {
			return microerror.Mask(err)
		}

		var values map[string]string
		err = yaml.Unmarshal([]byte(cm.Data["values"]), &values)
		if err != nil {
			return microerror.Mask(err)
		}

		v := Values{
			ClusterDNSIP: values["clusterDNSIP"],
			Image: Image{
				Registry: r.registryDomain,
			},
			Tiller: Tiller{
				Namespace: chartOperatorNamespace,
			},
		}

		chartOperatorValue, err = json.Marshal(v)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	{
		err = helmClient.InstallReleaseFromTarball(ctx, tarballPath, chartOperatorNamespace, helm.ReleaseName(chartOperatorRelease), helm.ValueOverrides(chartOperatorValue))
		if err != nil {
			return microerror.Mask(err)
		}
	}

	return nil
}
