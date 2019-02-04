package secret

import (
	"context"
	"strings"

	"github.com/giantswarm/apiextensions/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/microerror"
	yaml "gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/app-operator/pkg/label"
	"github.com/giantswarm/app-operator/service/controller/app/v1/key"
	appcatalogkey "github.com/giantswarm/app-operator/service/controller/appcatalog/v1/key"
)

func (r *Resource) GetDesiredState(ctx context.Context, obj interface{}) (interface{}, error) {
	cr, err := key.ToCustomResource(obj)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	values, err := r.getValuesYaml(ctx, cr)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:        key.ConfigMapName(cr),
			Namespace:   key.Namespace(cr),
			Annotations: cr.Annotations,
			Labels:      processLabels(r.projectName, cr.Labels),
		},
		Data: map[string][]byte{
			valuesKey: values,
		},
	}

	return secret, nil
}

func (r *Resource) getValuesYaml(ctx context.Context, cr v1alpha1.App) ([]byte, error) {
	valuesYaml := []byte{}

	appCatalogValues, err := r.getAppCatalogValues(ctx, cr)
	if err != nil {
		return valuesYaml, microerror.Mask(err)
	}

	appValues, err := r.getAppValues(ctx, cr)
	if err != nil {
		return valuesYaml, microerror.Mask(err)
	}

	values, err := union(appCatalogValues, appValues)
	if err != nil {
		return valuesYaml, microerror.Mask(err)
	}

	valuesYaml, err = yaml.Marshal(values)
	if err != nil {
		return valuesYaml, microerror.Mask(err)
	}

	return valuesYaml, nil
}

func (r *Resource) getAppCatalogValues(ctx context.Context, cr v1alpha1.App) (map[string]interface{}, error) {
	values := make(map[string]interface{})

	catalogName := key.CatalogName(cr)

	appCatalog, err := r.g8sClient.ApplicationV1alpha1().AppCatalogs(r.watchNamespace).Get(catalogName, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return nil, microerror.Maskf(notFoundError, "appCatalog %#q in namespace %#q", catalogName, "default")
	} else if err != nil {
		return nil, microerror.Mask(err)
	}

	if appcatalogkey.SecretName(*appCatalog) != "" {
		values, err := r.getSecretValues(ctx, appcatalogkey.SecretName(*appCatalog), appcatalogkey.SecretNamespace(*appCatalog))
		if err != nil {
			return values, microerror.Mask(err)
		}
	}

	return values, nil
}

func (r *Resource) getAppValues(ctx context.Context, cr v1alpha1.App) (map[string]interface{}, error) {
	values := make(map[string]interface{})

	if key.SecretName(cr) != "" {
		values, err := r.getSecretValues(ctx, key.SecretName(cr), key.SecretNamespace(cr))
		if err != nil {
			return values, microerror.Mask(err)
		}
	}

	return values, nil
}

func (r *Resource) getSecretValues(ctx context.Context, secretName, secretNamespace string) (map[string]interface{}, error) {
	secretValues := make(map[string]interface{})

	secret, err := r.k8sClient.CoreV1().Secrets(secretNamespace).Get(secretName, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return nil, microerror.Maskf(notFoundError, "secret %#q in namespace %#q not found", secretName, secretNamespace)
	} else if err != nil {
		return nil, microerror.Mask(err)
	}

	yamlData := secret.Data[valuesKey]
	if yamlData != nil {
		err = yaml.Unmarshal(yamlData, &secretValues)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	return secretValues, nil
}

func processLabels(projectName string, inputLabels map[string]string) map[string]string {
	labels := map[string]string{
		label.ManagedBy: projectName,
	}

	for k, v := range inputLabels {
		if strings.HasPrefix(k, label.GiantSwarmPrefix) && k != label.ManagedBy {
			labels[k] = v
		} else if k == label.App {
			labels[k] = v
		}
	}

	return labels
}

func union(a, b map[string]interface{}) (map[string]interface{}, error) {
	if a == nil {
		return b, nil
	}

	for k, v := range b {
		_, ok := a[k]
		if ok {
			// The values have at least one shared key. We cannot decide which
			// value should be applied.
			return nil, microerror.Maskf(failedExecutionError, "values share the same key %#q", k)
		}
		a[k] = v
	}
	return a, nil
}
