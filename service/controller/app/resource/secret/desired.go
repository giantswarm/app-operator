package secret

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

	"github.com/giantswarm/app-operator/v6/pkg/project"
	"github.com/giantswarm/app-operator/v6/pkg/status"
	"github.com/giantswarm/app-operator/v6/service/controller/app/controllercontext"
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
		// Return empty chart secret so it is deleted.
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      key.ChartSecretName(cr),
				Namespace: r.chartNamespace,
			},
		}

		return secret, nil
	}

	// If no user-provided secret name is present, check if a *-user-values config map exists and set the reference
	if key.UserSecretName(cr) == "" {
		userSec, err := cc.Clients.K8s.K8sClient().CoreV1().Secrets(r.chartNamespace).Get(ctx, fmt.Sprintf("%s-user-secrets", cr.Name), metav1.GetOptions{})
		if err == nil {
			cr.Spec.UserConfig.Secret.Name = userSec.GetName()
			cr.Spec.UserConfig.Secret.Namespace = userSec.GetNamespace()
			err = cc.Clients.K8s.CtrlClient().Update(ctx, &cr)
			if err != nil {
				return nil, microerror.Mask(err)
			}
		}
	}

	mergedData, err := r.values.MergeSecretData(ctx, cr, cc.Catalog)
	if values.IsNotFound(err) {
		r.logger.LogCtx(ctx, "level", "warning", "message", "dependent secrets are not found")
		addStatusToContext(cc, err.Error(), status.SecretMergeFailedStatus)

		r.logger.Debugf(ctx, "canceling resource")
		resourcecanceledcontext.SetCanceled(ctx)
		return nil, nil
	} else if values.IsParsingError(err) {
		r.logger.LogCtx(ctx, "level", "warning", "message", "failed to merging secrets")
		addStatusToContext(cc, err.Error(), status.SecretMergeFailedStatus)

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

	secret := &corev1.Secret{
		Data: map[string][]byte{
			"values": bytes,
		},
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
