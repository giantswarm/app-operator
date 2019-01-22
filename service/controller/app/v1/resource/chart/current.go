package chart

import (
	"context"
	"fmt"

	"github.com/giantswarm/microerror"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/app-operator/service/controller/app/v1/key"
)

func (r *Resource) GetCurrentState(ctx context.Context, obj interface{}) (interface{}, error) {
	customResource, err := key.ToCustomResource(obj)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	name := key.AppName(customResource)
	client, err := r.kubeConfig.NewG8sClientForApp(ctx, customResource)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	chart, err := client.ApplicationV1alpha1().Charts(r.watchNamespace).Get(name, metav1.GetOptions{})

	if errors.IsNotFound(err) {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("did not find chart %#q", name))
		return nil, nil
	} else if err != nil {
		return nil, microerror.Mask(err)
	}

	return chart, nil
}
