package configmap

import (
	"context"

	"github.com/giantswarm/microerror"
	yaml "gopkg.in/yaml.v2"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/app-operator/service/controller/app/v1/appvalues"
	"github.com/giantswarm/app-operator/service/controller/app/v1/key"
	appcatalogkey "github.com/giantswarm/app-operator/service/controller/appcatalog/v1/key"
)

func (r *Resource) GetDesiredState(ctx context.Context, obj interface{}) (interface{}, error) {
	var err error

	cr, err := key.ToCustomResource(obj)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	catalogName := key.CatalogName(cr)

	appCatalog, err := r.g8sClient.ApplicationV1alpha1().AppCatalogs(r.watchNamespace).Get(catalogName, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return nil, microerror.Maskf(notFoundError, "appCatalog %#q in namespace %#q", catalogName, r.watchNamespace)
	} else if err != nil {
		return nil, microerror.Mask(err)
	}

	appCatalogValues := make(map[string]interface{})
	catalogConfigMapName := appcatalogkey.ConfigMapName(*appCatalog)

	if catalogConfigMapName != "" {
		catalogConfigMapNamespace := appcatalogkey.ConfigMapNamespace(*appCatalog)
		appCatalogValues, err = r.getConfigMapValues(ctx, catalogConfigMapName, catalogConfigMapNamespace)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	appValues := make(map[string]interface{})
	appConfigMapName := key.ConfigMapName(cr)

	if appConfigMapName != "" {
		appConfigMapNamespace := key.ConfigMapNamespace(cr)
		appValues, err = r.getConfigMapValues(ctx, appConfigMapName, appConfigMapNamespace)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	values, err := appvalues.MergeValues(appValues, appCatalogValues)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	_, err = yaml.Marshal(values)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return nil, nil
}

func (r *Resource) getConfigMapValues(ctx context.Context, configMapName, configMapNamespace string) (map[string]interface{}, error) {
	values := make(map[string]interface{})

	configMap, err := r.k8sClient.CoreV1().ConfigMaps(configMapNamespace).Get(configMapName, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return nil, microerror.Maskf(notFoundError, "configmap %#q in namespace %#q", configMapName, configMapNamespace)
	} else if err != nil {
		return nil, microerror.Mask(err)
	}

	yamlValues, ok := configMap.Data[valuesKey]
	if !ok {
		return values, nil
	}

	data := []byte(yamlValues)
	err = yaml.Unmarshal(data, &values)
	if err != nil {
		return values, microerror.Mask(err)
	}

	return values, nil
}
