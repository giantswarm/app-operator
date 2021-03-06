package releasemigration

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/giantswarm/apiextensions/v3/pkg/clientset/versioned"
	"github.com/giantswarm/app/v5/pkg/key"
	"github.com/giantswarm/k8sclient/v5/pkg/k8sclient"
	"github.com/giantswarm/k8smetadata/pkg/annotation"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/v5/pkg/controller/context/reconciliationcanceledcontext"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/giantswarm/app-operator/v5/service/controller/app/controllercontext"
)

// EnsureCreated ensures helm release is migrated from a v2 configmap to a v3 secret.
func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	cr, err := key.ToApp(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	if cc.Status.ClusterStatus.IsUnavailable {
		r.logger.Debugf(ctx, "workload cluster is unavailable")
		r.logger.Debugf(ctx, "canceling resource")
		return nil
	}

	// Resource is used to migrating Helm 2 release into Helm 3 in case of chart-operator app reconciliation.
	// So for other apps we can skip this step.
	if key.AppName(cr) != key.ChartOperatorAppName {
		r.logger.Debugf(ctx, "no need to migrate release for %#q", key.AppName(cr))
		r.logger.Debugf(ctx, "canceling resource")
		return nil
	}

	if key.Version(cr) != cr.Status.Version {
		r.logger.Debugf(ctx, "app %#q is not reconciled to the latest desired status yet", key.AppName(cr))
		r.logger.Debugf(ctx, "canceling resource")
		return nil
	}

	if cr.Status.AppVersion == "" {
		r.logger.Debugf(ctx, "app %#q is not installed yet", key.AppName(cr))
	}

	v, err := semver.NewVersion(cr.Status.AppVersion)
	if err != nil {
		return microerror.Mask(err)
	}

	if v.Major() < 1 {
		r.logger.Debugf(ctx, "app %#q with appVersion %#q is using Helm 2. we don't need to trigger Helm 3 migration.", key.AppName(cr), cr.Status.AppVersion)
		r.logger.Debugf(ctx, "canceling resource")
		return nil
	}

	deploy, err := cc.Clients.K8s.K8sClient().AppsV1().Deployments(key.Namespace(cr)).Get(ctx, cr.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		r.logger.Debugf(ctx, "app %#q has no deployement object yet", key.AppName(cr))
		r.logger.Debugf(ctx, "canceling resource")
		return nil
	} else if err != nil {
		return microerror.Mask(err)
	}

	// extract spec container image
	image := deploy.Spec.Template.Spec.Containers[0].Image
	tag := strings.Split(image, ":")[1]

	v, err = semver.NewVersion(tag)
	if err != nil {
		return microerror.Mask(err)
	}

	if v.Major() < 1 {
		r.logger.Debugf(ctx, "app %#q with appVersion %#q is using Helm 2. we don't need to trigger Helm 3 migration.", key.AppName(cr), cr.Status.AppVersion)
		r.logger.Debugf(ctx, "canceling resource")
		return nil
	}

	if deploy.Status.ReadyReplicas == 0 {
		r.logger.Debugf(ctx, "app %#q is not deployed yet", key.AppName(cr))
		r.logger.Debugf(ctx, "canceling resource")
		return nil
	}

	var tillerNamespace string
	{
		if key.InCluster(cr) {
			tillerNamespace = metav1.NamespaceSystem
		} else {
			tillerNamespace = r.chartNamespace
		}
	}

	hasConfigMap, err := r.hasHelmV2ConfigMaps(ctx, cc.Clients.K8s, tillerNamespace)
	if err != nil {
		return microerror.Mask(err)
	}

	hasSecret, err := r.hasHelmV3Secrets(ctx, cc.Clients.K8s)
	if err != nil {
		return microerror.Mask(err)
	}

	// If Helm v2 release configmap had not been deleted and Helm v3 release secret is there,
	// It means helm release migration is in progress.
	if hasConfigMap && hasSecret {
		r.logger.Debugf(ctx, "release %#q helmV3 migration in progress", key.ReleaseName(cr))

		found, err := findMigrationApp(ctx, cc.Clients.Helm, tillerNamespace)
		if err != nil {
			return microerror.Mask(err)
		}

		if !found {
			r.logger.Debugf(ctx, "release %#q had been purged during migration, reinstalling...", migrationApp)
			err = r.ensureReleasesMigrated(ctx, cc.Clients.K8s, cc.Clients.Helm, tillerNamespace)
			if err != nil {
				return microerror.Mask(err)
			}
			r.logger.Debugf(ctx, "installed %#q", migrationApp)
		}

		r.logger.Debugf(ctx, "canceling reconciliation")
		reconciliationcanceledcontext.SetCanceled(ctx)
		return nil
	}

	// If Helm v2 release configmap had not been deleted and Helm v3 release secret was not created,
	// It means helm v3 release migration is not started.
	if hasConfigMap && !hasSecret {
		r.logger.Debugf(ctx, "release %#q helmV3 migration not started", key.ReleaseName(cr))
		r.logger.Debugf(ctx, "installing %#q", migrationApp)

		// cordon all charts except chart-operator
		err := r.cordonChart(ctx, cc.Clients.K8s.G8sClient())
		if err != nil {
			return microerror.Mask(err)
		}

		// install helm-2to3-migration app
		err = r.ensureReleasesMigrated(ctx, cc.Clients.K8s, cc.Clients.Helm, tillerNamespace)
		if IsReleaseAlreadyExists(err) {
			r.logger.Debugf(ctx, "release %#q already exists", migrationApp)
			r.logger.Debugf(ctx, "canceling reconciliation")
			reconciliationcanceledcontext.SetCanceled(ctx)
			return nil
		} else if err != nil {
			return microerror.Mask(err)
		}

		r.logger.Debugf(ctx, "installed %#q", migrationApp)
		return nil
	}

	// If Helm v2 release configmap had been deleted and Helm v3 release secret was created,
	// It means helm v3 release migration is finished.
	if !hasConfigMap && hasSecret {
		err = r.uncordonChart(ctx, cc.Clients.K8s.G8sClient())
		if err != nil {
			return microerror.Mask(err)
		}
		err = r.deleteMigrationApp(ctx, cc.Clients.Helm, tillerNamespace)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	r.logger.Debugf(ctx, "no pending migration for release %#q", key.ReleaseName(cr))

	return nil
}

func (r *Resource) cordonChart(ctx context.Context, g8sClient versioned.Interface) error {
	lo := metav1.ListOptions{
		FieldSelector: "metadata.name!=chart-operator-unique",
		LabelSelector: "app notin (chart-operator)",
	}
	charts, err := g8sClient.ApplicationV1alpha1().Charts(r.chartNamespace).List(ctx, lo)
	if err != nil {
		return microerror.Mask(err)
	}

	r.logger.Debugf(ctx, "cordoning %d charts", len(charts.Items))

	cordonReason := replaceToEscape(annotation.ChartOperatorCordonReason)
	cordonUntil := replaceToEscape(annotation.ChartOperatorCordonUntil)

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

		_, err = g8sClient.ApplicationV1alpha1().Charts(chart.Namespace).Patch(ctx, chart.Name, types.JSONPatchType, bytes, metav1.PatchOptions{})
		if err != nil {
			return microerror.Mask(err)
		}
	}
	r.logger.Debugf(ctx, "cordoned %d charts", len(charts.Items))

	return nil
}

func (r *Resource) uncordonChart(ctx context.Context, g8sClient versioned.Interface) error {
	lo := metav1.ListOptions{
		FieldSelector: "metadata.name!=chart-operator-unique",
		LabelSelector: "app notin (chart-operator)",
	}
	charts, err := g8sClient.ApplicationV1alpha1().Charts(r.chartNamespace).List(ctx, lo)
	if err != nil {
		return microerror.Mask(err)
	}
	r.logger.Debugf(ctx, "uncordoning cordoned charts")

	cordonReason := replaceToEscape(annotation.ChartOperatorCordonReason)
	cordonUntil := replaceToEscape(annotation.ChartOperatorCordonUntil)
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
		_, err = g8sClient.ApplicationV1alpha1().Charts(chart.Namespace).Patch(ctx, chart.Name, types.JSONPatchType, bytes, metav1.PatchOptions{})
		if err != nil {
			return microerror.Mask(err)
		}
		i++
	}
	r.logger.Debugf(ctx, "uncordoned %d charts", i)

	return nil
}

func (r *Resource) hasHelmV2ConfigMaps(ctx context.Context, k8sClient k8sclient.Interface, tillerNamespace string) (bool, error) {
	chartMap, err := getChartMap(ctx, k8sClient, r.chartNamespace)
	if err != nil {
		return false, microerror.Mask(err)
	}

	lo := metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s", "OWNER", "TILLER"),
	}

	// Check whether helm 2 release configMaps still exist.
	cms, err := k8sClient.K8sClient().CoreV1().ConfigMaps(tillerNamespace).List(ctx, lo)
	if err != nil {
		return false, microerror.Mask(err)
	}

	var count int
	for _, cm := range cms.Items {
		if _, ok := chartMap[cm.GetLabels()["NAME"]]; !ok {
			continue
		}
		count++
	}

	return count > 0, nil
}

func (r *Resource) hasHelmV3Secrets(ctx context.Context, k8sClient k8sclient.Interface) (bool, error) {
	var releaseNamespaces []string
	{
		list, err := k8sClient.G8sClient().ApplicationV1alpha1().Charts(r.chartNamespace).List(ctx, metav1.ListOptions{})
		if err != nil {
			return false, microerror.Mask(err)
		}

		namespaces := map[string]bool{}
		for _, chart := range list.Items {
			namespaces[chart.Spec.Namespace] = true
		}

		for ns := range namespaces {
			releaseNamespaces = append(releaseNamespaces, ns)
		}
	}

	lo := metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s", "owner", "helm"),
	}
	var length int
	// Check whether helm 3 release secret exists.
	for _, namespace := range releaseNamespaces {
		secrets, err := k8sClient.K8sClient().CoreV1().Secrets(namespace).List(ctx, lo)
		if err != nil {
			return false, microerror.Mask(err)
		}

		length += len(secrets.Items)
	}

	return length > 0, nil
}

func replaceToEscape(from string) string {
	return strings.Replace(from, "/", "~1", -1)
}

func checkMigrationJobStatus(ctx context.Context, k8sClient k8sclient.Interface, releaseNamespace string) (bool, error) {
	job, err := k8sClient.K8sClient().BatchV1().Jobs(releaseNamespace).Get(ctx, migrationApp, metav1.GetOptions{})
	if err != nil {
		return false, microerror.Mask(err)
	}

	return job.Status.Succeeded > 0, nil
}

func getChartMap(ctx context.Context, k8sClient k8sclient.Interface, namespace string) (map[string]bool, error) {
	charts := make(map[string]bool)

	// Get list of chart CRs as not all helm 2 releases will have a chart CR.
	list, err := k8sClient.G8sClient().ApplicationV1alpha1().Charts(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, microerror.Mask(err)
	}

	for _, chart := range list.Items {
		charts[chart.Name] = true
	}

	return charts, nil
}
