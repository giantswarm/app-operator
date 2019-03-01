package index

import (
	"context"
	"fmt"

	"github.com/giantswarm/microerror"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/app-operator/service/controller/appcatalog/v1/key"
)

func (r *StateGetter) GetCurrentState(ctx context.Context, obj interface{}) ([]*v1.ConfigMap, error) {
	cr, err := key.ToCustomResource(obj)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	name := configMapName(cr.Name)

	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("finding index configMap %#q", name))

	cm, err := r.k8sClient.CoreV1().ConfigMaps(r.indexNamespace).Get(name, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("did not find index configMap %#q", name))
		return nil, nil
	} else if err != nil {
		return nil, microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("found index configMap %#q", name))

	return []*v1.ConfigMap{cm}, nil
}
