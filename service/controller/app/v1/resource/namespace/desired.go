package namespace

import (
	"context"
	"fmt"

	"github.com/giantswarm/microerror"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/app-operator/pkg/label"
	"github.com/giantswarm/app-operator/pkg/project"
	"github.com/giantswarm/app-operator/service/controller/app/v1/key"
)

func (r *Resource) GetDesiredState(ctx context.Context, obj interface{}) (interface{}, error) {
	cr, err := key.ToCustomResource(obj)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	var clusterID, org string
	{
		name := key.ClusterValuesConfigMapName(cr)
		cm, err := r.k8sClient.CoreV1().ConfigMaps(cr.Namespace).Get(name, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			r.logger.LogCtx(ctx, "level", "warning", "message", fmt.Sprintf("no cluster-value %#q in control plane, operator will use empty value for namespace", name))
		} else if err != nil {
			return nil, microerror.Mask(err)
		} else {
			clusterID = cm.GetLabels()[label.Cluster]
			org = cm.GetLabels()[label.Organization]
		}
	}

	// Compute the desired state of the namespace to have a reference of how
	// the data should be.
	namespace := &corev1.Namespace{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Namespace",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: namespaceName,
			Labels: map[string]string{
				label.Cluster:      clusterID,
				label.ManagedBy:    project.Name(),
				label.Organization: org,
			},
		},
	}

	return namespace, nil
}
