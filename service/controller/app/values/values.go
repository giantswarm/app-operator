package values

import (
	"context"

	"github.com/giantswarm/apiextensions/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/helmclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	yaml "gopkg.in/yaml.v2"
	"k8s.io/client-go/kubernetes"
)

// Config represents the configuration used to create a new values service.
type Config struct {
	// Dependencies.
	K8sClient kubernetes.Interface
	Logger    micrologger.Logger
}

// Values implements the values service.
type Values struct {
	// Dependencies.
	k8sClient kubernetes.Interface
	logger    micrologger.Logger
}

// New creates a new configured values service.
func New(config Config) (*Values, error) {
	if config.K8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.K8sClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	r := &Values{
		// Dependencies.
		k8sClient: config.K8sClient,
		logger:    config.Logger,
	}

	return r, nil
}

// MergeAll merges both configmap and secret values to produce a single set of
// values that can be passed to Helm.
func (v *Values) MergeAll(ctx context.Context, app v1alpha1.App, appCatalog v1alpha1.AppCatalog) (map[string]interface{}, error) {
	configMapData, err := v.MergeConfigMapData(ctx, app, appCatalog)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	secretData, err := v.MergeSecretData(ctx, app, appCatalog)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	values, err := helmclient.MergeValues(toByteSliceMap(configMapData), secretData)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return values, nil
}

// mergeData contains the shared logic that is common to merging configmap and
// secret data.
func mergeData(destMap, srcMap map[string][]byte) (map[string][]byte, error) {
	mergedData, err := helmclient.MergeValues(destMap, srcMap)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	bytes, err := yaml.Marshal(mergedData)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	// Key name is just for display purposes. The only restriction is the
	// configmap or secret storing data for the chart CR can only have a single
	// key with YAML values.
	result := map[string][]byte{
		"values": bytes,
	}

	return result, nil
}

// toByteSliceMap converts from a string map to a byte slice map.
func toByteSliceMap(input map[string]string) map[string][]byte {
	result := map[string][]byte{}

	for k, v := range input {
		result[k] = []byte(v)
	}

	return result
}

// toStringMap converts from a byte slice map to a string map.
func toStringMap(input map[string][]byte) map[string]string {
	result := map[string]string{}

	for k, v := range input {
		result[k] = string(v)
	}

	return result
}
