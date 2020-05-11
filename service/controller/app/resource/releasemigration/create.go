package releasemigration

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/giantswarm/apiextensions/pkg/clientset/versioned"
	"github.com/giantswarm/helmclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/controller/context/reconciliationcanceledcontext"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"

	"github.com/giantswarm/app-operator/pkg/annotation"
	"github.com/giantswarm/app-operator/service/controller/app/controllercontext"
	"github.com/giantswarm/app-operator/service/controller/app/key"
)

// EnsureCreated ensures helm release is migrated from a v2 configmap to a v3 secret.
func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	cr, err := key.ToCustomResource(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	if cc.Status.TenantCluster.IsUnavailable {
		r.logger.LogCtx(ctx, "level", "debug", "message", "tenant cluster is unavailable")
		r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")
		return nil
	}

	//TODO: Remove this statement when we need to test on control planes.
	/*
		if key.InCluster(cr) {
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("app %#q uses InCluster kubeconfig no need to migrate releases", key.AppName(cr)))
			r.logger.LogCtx(ctx, "level", "debug", "message", "cancelling the resource")
			return nil
		}
	*/

	// Resource is used to migrating Helm 2 release into Helm 3 in case of chart-operator app reconciliation.
	// So for other apps we can skip this step.
	if key.AppName(cr) != key.ChartOperatorAppName {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("no need to migrating Helm release for %#q", key.AppName(cr)))
		r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")
		return nil
	}

	if cr.Status.AppVersion == "" {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("app %#q is not installed yet", key.AppName(cr)))
		r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")
		return nil
	}

	v, err := semver.NewVersion(cr.Status.AppVersion)
	if err != nil {
		return microerror.Mask(err)
	}

	if v.Major() < 1 {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("app %#q with appVersion %#q is using Helm 2. we don't need to trigger Helm 3 migration.", key.AppName(cr), cr.Status.AppVersion))
		r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")
		return nil
	}

	if strings.ToLower(cr.Status.Release.Status) != helmclient.StatusDeployed {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("app %#q is not deployed yet", key.AppName(cr)))
		r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")
		return nil
	}

	var tillerNamespace string
	{
		if key.InCluster(cr) {
			tillerNamespace = metav1.NamespaceSystem
		} else {
			tillerNamespace = "giantswarm"
		}
	}

	hasConfigMap, err := r.hasHelmV2ConfigMaps(cc.Clients.K8s.K8sClient(), key.ReleaseName(cr), tillerNamespace)
	if err != nil {
		return microerror.Mask(err)
	}

	hasSecret, err := r.hasHelmV3Secrets(cc.Clients.K8s.K8sClient(), key.ReleaseName(cr), key.Namespace(cr))
	if err != nil {
		return microerror.Mask(err)
	}

	// If Helm v2 release configmap had not been deleted and Helm v3 release secret is there,
	// It means helm release migration is in progress.
	if hasConfigMap && hasSecret {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("release %#q helmV3 migration in progress", key.ReleaseName(cr)))
		r.logger.LogCtx(ctx, "level", "debug", "message", "canceling reconciliation")
		reconciliationcanceledcontext.SetCanceled(ctx)
		return nil
	}

	// If Helm v2 release configmap had not been deleted and Helm v3 release secret was not created,
	// It means helm v3 release migration is not started.
	if hasConfigMap && !hasSecret {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("release %#q helmV3 migration not started", key.ReleaseName(cr)))
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("installing %#q", migrationApp))

		// cordon all charts except chart-operator
		err := r.cordonChart(ctx, cc.Clients.K8s.G8sClient())
		if err != nil {
			return microerror.Mask(err)
		}

		// install helm-2to3-migration app
		err = r.ensureReleasesMigrated(ctx, cc.Clients.K8s.K8sClient(), cc.Clients.Helm, tillerNamespace)
		if IsReleaseAlreadyExists(err) {
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("release %#q already exists", migrationApp))
			r.logger.LogCtx(ctx, "level", "debug", "message", "canceling reconciliation")
			reconciliationcanceledcontext.SetCanceled(ctx)
			return nil
		} else if err != nil {
			return microerror.Mask(err)
		}

		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("installed %#q", migrationApp))
		return nil
	}

	// If Helm v2 release configmap had been deleted and Helm v3 release secret was created,
	// It means helm v3 release migration is finished.
	if !hasConfigMap && hasSecret {
		err = r.uncordonChart(ctx, cc.Clients.K8s.G8sClient())
		if err != nil {
			return microerror.Mask(err)
		}
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("no pending migration for release %#q", key.ReleaseName(cr)))

	return nil
}

func (r *Resource) cordonChart(ctx context.Context, g8sClient versioned.Interface) error {
	lo := metav1.ListOptions{
		LabelSelector: "app notin (chart-operator)",
	}
	charts, err := g8sClient.ApplicationV1alpha1().Charts("giantswarm").List(lo)
	if err != nil {
		return microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("cordoning %d charts", len(charts.Items)))

	cordonReason := replaceToEscape(fmt.Sprintf("%s/%s", annotation.ChartOperatorPrefix, annotation.CordonReason))
	cordonUntil := replaceToEscape(fmt.Sprintf("%s/%s", annotation.ChartOperatorPrefix, annotation.CordonUntil))

	for _, chart := range charts.Items {
		patches := []patch{}

		if len(chart.Annotations) == 0 {
			patches = append(patches, patch{
				Op:    "add",
				Path:  "/metadata/annotations",
				Value: map[string]string{},
			})
		}

		patches = append(patches, []patch{
			{
				Op:    "add",
				Path:  fmt.Sprintf("/metadata/annotations/%s", cordonReason),
				Value: "Migrating to helm 3",
			},
			{
				Op:    "add",
				Path:  fmt.Sprintf("/metadata/annotations/%s", cordonUntil),
				Value: key.CordonUntilDate(),
			},
		}...)

		bytes, err := json.Marshal(patches)
		if err != nil {
			return microerror.Mask(err)
		}

		_, err = g8sClient.ApplicationV1alpha1().Charts(chart.Namespace).Patch(chart.Name, types.JSONPatchType, bytes)
		if err != nil {
			return microerror.Mask(err)
		}
	}
	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("cordoned %d charts", len(charts.Items)))

	return nil
}

func (r *Resource) uncordonChart(ctx context.Context, g8sClient versioned.Interface) error {
	lo := metav1.ListOptions{
		LabelSelector: "app notin (chart-operator)",
	}
	charts, err := g8sClient.ApplicationV1alpha1().Charts("giantswarm").List(lo)
	if err != nil {
		return microerror.Mask(err)
	}
	r.logger.LogCtx(ctx, "level", "debug", "message", "uncordoning cordoned charts")

	cordonReason := replaceToEscape(fmt.Sprintf("%s/%s", annotation.ChartOperatorPrefix, annotation.CordonReason))
	cordonUntil := replaceToEscape(fmt.Sprintf("%s/%s", annotation.ChartOperatorPrefix, annotation.CordonUntil))
	patches := []patch{
		{
			Op:   "remove",
			Path: fmt.Sprintf("/metadata/annotations/%s", cordonReason),
		},
		{
			Op:   "remove",
			Path: fmt.Sprintf("/metadata/annotations/%s", cordonUntil),
		},
	}

	bytes, err := json.Marshal(patches)
	if err != nil {
		return microerror.Mask(err)
	}

	i := 0
	for _, chart := range charts.Items {
		if !key.IsChartCordoned(chart) {
			continue
		}
		_, err = g8sClient.ApplicationV1alpha1().Charts(chart.Namespace).Patch(chart.Name, types.JSONPatchType, bytes)
		if err != nil {
			return microerror.Mask(err)
		}
		i++
	}
	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("uncordoned %d charts", i))

	return nil
}

func (r *Resource) hasHelmV2ConfigMaps(k8sClient kubernetes.Interface, releaseName, tillerNamespace string) (bool, error) {
	lo := metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s,%s=%s", "NAME", releaseName, "OWNER", "TILLER"),
	}

	// Check whether helm 2 release configMaps still exist.
	cms, err := k8sClient.CoreV1().ConfigMaps(tillerNamespace).List(lo)
	if err != nil {
		return false, microerror.Mask(err)
	}

	return len(cms.Items) > 0, nil
}

func (r *Resource) hasHelmV3Secrets(k8sClient kubernetes.Interface, releaseName, releaseNamespace string) (bool, error) {
	lo := metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s,%s=%s", "name", releaseName, "owner", "helm"),
	}

	// Check whether helm 3 release secret exists.
	secrets, err := k8sClient.CoreV1().Secrets(releaseNamespace).List(lo)
	if err != nil {
		return false, microerror.Mask(err)
	}

	return len(secrets.Items) > 0, nil
}

func replaceToEscape(from string) string {
	return strings.Replace(from, "/", "~1", -1)
}
