package chart

import (
	"context"
	"fmt"
	"github.com/giantswarm/app-operator/service/controller/app/v1/key"
	"github.com/giantswarm/microerror"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (r *Resource) GetCurrentState(ctx context.Context, obj interface{}) (interface{}, error) {
	customResource, err := key.ToCustomResource(obj)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	name := key.AppName(customResource)
	chart, err := r.g8sClient.ApplicationV1alpha1().Charts(r.watchNamespace).Get(name, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("chart %#q is not created yet", name))
			return nil, nil
		}
		return nil, microerror.Mask(err)
	}

	return chart, nil
}
