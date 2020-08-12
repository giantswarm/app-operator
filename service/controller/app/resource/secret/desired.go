package secret

import (
	"context"
	"fmt"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/v2/pkg/controller/context/resourcecanceledcontext"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/app-operator/v2/pkg/annotation"
	"github.com/giantswarm/app-operator/v2/pkg/label"
	"github.com/giantswarm/app-operator/v2/pkg/project"
	"github.com/giantswarm/app-operator/v2/pkg/status"
	"github.com/giantswarm/app-operator/v2/service/controller/app/controllercontext"
	"github.com/giantswarm/app-operator/v2/service/controller/app/key"
	"github.com/giantswarm/app-operator/v2/service/controller/app/values"
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

	if key.IsDeleted(cr) {
		// Return empty chart secret so it is deleted.
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      key.ChartSecretName(cr),
				Namespace: r.chartNamespace,
			},
		}

		return secret, nil
	}

	mergedData, err := r.values.MergeSecretData(ctx, cr, cc.AppCatalog)
	if values.IsNotFound(err) {
		r.logger.LogCtx(ctx, "level", "warning", "message", "dependent secrets are not found")
		addStatusToContext(cc, err.Error(), status.SecretMergeFailedStatus)

		r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")
		resourcecanceledcontext.SetCanceled(ctx)
		return nil, nil
	} else if values.IsParsingError(err) {
		r.logger.LogCtx(ctx, "level", "warning", "message", "failed to merging secrets")
		addStatusToContext(cc, err.Error(), status.SecretMergeFailedStatus)

		r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")
		resourcecanceledcontext.SetCanceled(ctx)
		return nil, nil
	} else if err != nil {
		return nil, microerror.Mask(err)
	}

	if mergedData == nil {
		// Return early.
		return nil, nil
	}

	secret := &corev1.Secret{
		Data: mergedData,
		ObjectMeta: metav1.ObjectMeta{
			Name:      key.ChartSecretName(cr),
			Namespace: r.chartNamespace,
			Annotations: map[string]string{
				annotation.Notes: fmt.Sprintf("DO NOT EDIT. Values managed by %s.", project.Name()),
			},
			Labels: map[string]string{
				label.ManagedBy: project.Name(),
			},
		},
	}

	return secret, nil
}
