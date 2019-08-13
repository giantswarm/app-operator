package cordonchart

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/controller/context/resourcecanceledcontext"
	"k8s.io/apimachinery/pkg/types"

	"github.com/giantswarm/app-operator/pkg/annotation"
	"github.com/giantswarm/app-operator/service/controller/app/v1/controllercontext"
	"github.com/giantswarm/app-operator/service/controller/app/v1/key"
)

type mergeSpec struct {
	Op    string            `json:"op"`
	Path  string            `json:"path"`
	Value map[string]string `json:"value"`
}

func (r Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	cr, err := key.ToCustomResource(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	if !key.IsCordoned(cr) {
		return nil
	}

	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	if cc.Status.TenantCluster.IsDeleting {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("namespace %#q is being deleted, no need to reconcile resource", cr.Namespace))

		resourcecanceledcontext.SetCanceled(ctx)
		r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")

		return nil
	}

	name := cr.GetName()

	var mergeByte []byte
	{
		merge := []mergeSpec{
			{
				Op:   "add",
				Path: "/metadata/annotations",
				Value: map[string]string{
					annotation.CordonReason: key.CordonReason(cr),
					annotation.CordonUntil:  key.CordonUntil(cr),
				},
			},
		}

		mergeByte, err = json.Marshal(merge)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	_, err = cc.G8sClient.ApplicationV1alpha1().Charts(r.chartNamespace).Patch(name, types.JSONPatchType, mergeByte)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
