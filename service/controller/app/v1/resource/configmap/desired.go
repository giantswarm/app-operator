package configmap

import (
	"context"
	"fmt"

	"github.com/giantswarm/microerror"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/app-operator/pkg/label"
	"github.com/giantswarm/app-operator/pkg/project"
	"github.com/giantswarm/app-operator/service/controller/app/v1/controllercontext"
	"github.com/giantswarm/app-operator/service/controller/app/v1/key"
	"github.com/giantswarm/app-operator/service/controller/app/v1/values"
)

func (r *Resource) GetDesiredState(ctx context.Context, obj interface{}) (interface{}, error) {
	cr, err := key.ToCustomResource(obj)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	mergedData, err := r.values.MergeConfigMapData(ctx, cr, cc.AppCatalog)
	if values.IsNotFound(err) {
		r.logger.LogCtx(ctx, "level", "warning", "message", fmt.Sprintf("dependent configMaps are not found"), "stack", fmt.Sprintf("%#v", err))
		return nil, nil
	} else if err != nil {
		return nil, microerror.Mask(err)
	}

	if mergedData == nil {
		// Return early.
		return nil, nil
	}

	configMap := &corev1.ConfigMap{
		Data: mergedData,
		ObjectMeta: metav1.ObjectMeta{
			Name:      key.ChartConfigMapName(cr),
			Namespace: r.chartNamespace,
			Labels: map[string]string{
				label.ManagedBy: project.Name(),
			},
		},
	}

	return configMap, nil
}
