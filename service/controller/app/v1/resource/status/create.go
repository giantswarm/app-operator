package status

import (
	"context"
	"fmt"
	"k8s.io/apimachinery/pkg/api/errors"

	"github.com/giantswarm/microerror"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/app-operator/service/controller/app/v1/controllercontext"
	"github.com/giantswarm/app-operator/service/controller/app/v1/key"
)

func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	cr, err := key.ToCustomResource(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	name := key.AppName(cr)

	ctlCtx, err := controllercontext.FromContext(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("finding status for chart %#q", name))

	chart, err := ctlCtx.G8sClient.ApplicationV1alpha1().Charts(r.watchNamespace).Get(name, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		return microerror.Maskf(notFoundError, "chart %#q in namespace %#q", name, r.watchNamespace)
	} else if err != nil {
		return microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("found status for chart %#q", name))

	if chart.Status.Status != "" && key.AppStatus(cr) != chart.Status.Status {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("setting app %#q status as %#q", name, chart.Status.Status))

		crCopy := cr.DeepCopy()
		crCopy.ResourceVersion = chart.GetResourceVersion()
		crCopy.Status.AppVersion = chart.Status.AppVersion
		crCopy.Status.LastDeployed = *chart.Status.LastDeployed.DeepCopy()
		crCopy.Status.Status = chart.Status.Status
		crCopy.Status.Version = chart.Status.Version

		_, err = r.g8sClient.ApplicationV1alpha1().Apps(cr.Namespace).UpdateStatus(crCopy)
		if err != nil {
			return microerror.Mask(err)
		}

		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("status set for app %#q", name))
	} else {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("status for chart %#q already set to %#q", name, chart.Status.Status))
	}

	return nil
}
