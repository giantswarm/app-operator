package secret

import (
	"context"
	"fmt"

	"github.com/giantswarm/microerror"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/app-operator/pkg/label"
	"github.com/giantswarm/app-operator/service/controller/app/v1/controllercontext"
	"github.com/giantswarm/app-operator/service/controller/app/v1/key"
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

	if apierrors.IsNotFound(err) {
		r.logger.LogCtx(ctx, "level", "error", "message", fmt.Sprintf("dependent configMaps are not found"), "stack", fmt.Sprintf("%#v", err))
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
			Labels: map[string]string{
				label.ManagedBy: r.projectName,
			},
		},
	}

	return secret, nil
}
