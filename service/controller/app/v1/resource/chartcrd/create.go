package chartcrd

import (
	"context"
	"time"

	"github.com/giantswarm/apiextensions/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/app-operator/service/controller/app/v1/key"
	"github.com/giantswarm/backoff"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/client/k8scrdclient"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"

	"github.com/giantswarm/app-operator/service/controller/app/v1/controllercontext"
)

func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	cr, err := key.ToCustomResource(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	secretName := key.SecretName(cr)

	if secretName != "" {
		r.logger.LogCtx(ctx, "level", "debug", "message", "ensuring chart crd creation on tenant cluster")

		ctlCtx, err := controllercontext.FromContext(ctx)
		if err != nil {
			return microerror.Mask(err)
		}

		k8sExtClient, err := apiextensionsclient.NewForConfig(ctlCtx.RESTConfig)
		if err != nil {
			return microerror.Mask(err)
		}

		c := k8scrdclient.Config{
			K8sExtClient: k8sExtClient,
			Logger:       r.logger,
		}

		crdClient, err := k8scrdclient.New(c)
		crdBackoff := backoff.NewMaxRetries(3, 1*time.Second)

		err = crdClient.EnsureCreated(ctx, v1alpha1.NewChartCRD(), crdBackoff)
		if IsNotEstablished(err) {
			r.logger.LogCtx(ctx, "level", "debug", "message", "chart crd in creation at the moment")
			return nil
		} else if err != nil {
			return microerror.Mask(err)
		}
		r.logger.LogCtx(ctx, "level", "debug", "message", "ensured chart crd creation on tenant cluster")
	} else {
		r.logger.LogCtx(ctx, "level", "debug", "message", "do not need to create chart crd on tenant cluster")
	}

	return nil
}
