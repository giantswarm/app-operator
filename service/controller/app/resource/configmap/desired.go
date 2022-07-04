package configmap

import (
	"context"
	"fmt"

	"github.com/giantswarm/app/v6/pkg/key"
	"github.com/giantswarm/app/v6/pkg/values"
	"github.com/giantswarm/k8smetadata/pkg/annotation"
	"github.com/giantswarm/k8smetadata/pkg/label"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/v6/pkg/controller/context/resourcecanceledcontext"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"

	"github.com/giantswarm/app-operator/v7/pkg/project"
	"github.com/giantswarm/app-operator/v7/pkg/status"
	"github.com/giantswarm/app-operator/v7/service/controller/app/controllercontext"
)

func (r *Resource) GetDesiredState(ctx context.Context, obj interface{}) (interface{}, error) {
	cr, err := key.ToApp(obj)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	if key.IsDeleted(cr) {
		// Return empty chart configmap so it is deleted.
		configMap := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      key.ChartConfigMapName(cr),
				Namespace: r.chartNamespace,
			},
		}

		return configMap, nil
	}

	mergedData, err := r.values.MergeConfigMapData(ctx, cr, cc.Catalog)
	if values.IsNotFound(err) {
		r.logger.LogCtx(ctx, "level", "warning", "message", "dependent configMaps are not found")
		addStatusToContext(cc, err.Error(), status.ConfigmapMergeFailedStatus)

		r.logger.Debugf(ctx, "canceling resource")
		resourcecanceledcontext.SetCanceled(ctx)
		return nil, nil
	} else if values.IsParsingError(err) {
		r.logger.LogCtx(ctx, "level", "warning", "message", "failed to merging configMaps")
		addStatusToContext(cc, err.Error(), status.ConfigmapMergeFailedStatus)

		r.logger.Debugf(ctx, "canceling resource")
		resourcecanceledcontext.SetCanceled(ctx)
		return nil, nil
	} else if err != nil {
		return nil, microerror.Mask(err)
	}

	if mergedData == nil {
		// Return early.
		return nil, nil
	}

	bytes, err := yaml.Marshal(mergedData)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	configMap := &corev1.ConfigMap{
		Data: map[string]string{
			"values": string(bytes),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      key.ChartConfigMapName(cr),
			Namespace: r.chartNamespace,
			Annotations: map[string]string{
				annotation.Notes: fmt.Sprintf("DO NOT EDIT. Values managed by %s.", project.Name()),
			},
			Labels: map[string]string{
				label.ManagedBy: project.Name(),
			},
		},
	}

	return configMap, nil
}
