package configmap

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/giantswarm/apiextensions/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/microerror"
	yaml "gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/app-operator/pkg/label"
	"github.com/giantswarm/app-operator/service/controller/app/v1/controllercontext"
	"github.com/giantswarm/app-operator/service/controller/app/v1/key"
	appcatalogkey "github.com/giantswarm/app-operator/service/controller/appcatalog/v1/key"
)

func (r *Resource) GetDesiredState(ctx context.Context, obj interface{}) (interface{}, error) {
	cr, err := key.ToCustomResource(obj)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	data, err := r.mergeConfigMapData(ctx, cr)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	configMap := &corev1.ConfigMap{
		Data: data,
		ObjectMeta: metav1.ObjectMeta{
			Name:        key.ConfigMapName(cr),
			Namespace:   key.Namespace(cr),
			Annotations: cr.Annotations,
			Labels:      processLabels(r.projectName, cr.Labels),
		},
	}

	return configMap, nil
}

func (r *Resource) mergeConfigMapData(ctx context.Context, cr v1alpha1.App) (map[string]string, error) {
	appData := map[string]string{}
	catalogData := map[string]string{}

	appConfigMap, err := r.getAppConfigMap(ctx, cr)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	if appConfigMap != nil {
		appData = appConfigMap.Data
	}

	catalogConfigMap, err := r.getCatalogConfigMap(ctx, cr)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	if catalogConfigMap != nil {
		catalogData = appConfigMap.Data
	}

	data, err := mergeData(appData, catalogData)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return data, nil
}

func (r *Resource) getAppConfigMap(ctx context.Context, cr v1alpha1.App) (*corev1.ConfigMap, error) {
	if key.ConfigMapName(cr) != "" {
		// Return early as no configmap configured.
		return nil, nil
	}

	configMap, err := r.getConfigMap(ctx, key.ConfigMapName(cr), key.ConfigMapNamespace(cr))
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return configMap, nil
}

func (r *Resource) getCatalogConfigMap(ctx context.Context, cr v1alpha1.App) (*corev1.ConfigMap, error) {
	ctlCtx, err := controllercontext.FromContext(ctx)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	configMapName := appcatalogkey.ConfigMapName(ctlCtx.AppCatalog)
	if configMapName == "" {
		// Return early as no configmap configured.
		return nil, nil
	}

	configMap, err := r.getConfigMap(ctx, configMapName, appcatalogkey.ConfigMapNamespace(ctlCtx.AppCatalog))
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return configMap, nil
}

func (r *Resource) getConfigMap(ctx context.Context, configMapName, configMapNamespace string) (*corev1.ConfigMap, error) {
	configMap, err := r.k8sClient.CoreV1().ConfigMaps(configMapNamespace).Get(configMapName, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return nil, microerror.Maskf(notFoundError, "config map %#q in namespace %#q not found", configMapName, configMapNamespace)
	} else if err != nil {
		return nil, microerror.Mask(err)
	}

	return configMap, nil
}

func mergeData(appData, catalogData map[string]string) (map[string]string, error) {
	var err error

	fmt.Fprintf(os.Stdout, "APP DATA %#v", appData)
	fmt.Fprintf(os.Stdout, "CATALOG DATA %#v", catalogData)

	appValues := map[string]interface{}{}
	catalogValues := map[string]interface{}{}

	appYaml := appData[valuesKey]
	if appYaml != "" {
		err = yaml.Unmarshal([]byte(appYaml), &appValues)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	fmt.Fprintf(os.Stdout, "APP VALUES %#v", catalogValues)

	catalogYaml := catalogData[valuesKey]
	if catalogYaml != "" {
		err = yaml.Unmarshal([]byte(catalogYaml), &catalogValues)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	fmt.Fprintf(os.Stdout, "CATALOG VALUES %#v", catalogValues)

	values, err := union(appValues, catalogValues)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	fmt.Fprintf(os.Stdout, "VALUES %#v", values)

	if reflect.DeepEqual(values, map[string]interface{}{}) {
		return nil, nil
	}

	valuesYaml, err := yaml.Marshal(values)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	fmt.Fprintf(os.Stderr, "VALUES YAML %s", valuesYaml)

	data := map[string]string{
		valuesKey: string(valuesYaml),
	}

	return data, nil
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
