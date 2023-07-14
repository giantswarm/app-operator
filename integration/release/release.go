//go:build k8srequired
// +build k8srequired

package release

import (
	"context"
	"fmt"
	"time"

	helmv2 "github.com/fluxcd/helm-controller/api/v2beta1"
	"github.com/giantswarm/apiextensions-application/api/v1alpha1"
	"github.com/giantswarm/appcatalog"
	"github.com/giantswarm/backoff"
	"github.com/giantswarm/helmclient/v4/pkg/helmclient"
	"github.com/giantswarm/k8sclient/v7/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/afero"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/yaml"
)

type Config struct {
	HelmClient helmclient.Interface
	K8sClient  k8sclient.Interface
	Logger     micrologger.Logger
}

type Release struct {
	helmClient helmclient.Interface
	k8sClient  k8sclient.Interface
	logger     micrologger.Logger
}

type AppConfiguration struct {
	AppName      string
	AppNamespace string
	AppValues    string
	AppVersion   string
	CatalogURL   string
}

func New(config Config) (*Release, error) {
	if config.HelmClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.HelmClient must not be empty", config)
	}
	if config.K8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.K8sClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	r := &Release{
		helmClient: config.HelmClient,
		k8sClient:  config.K8sClient,
		logger:     config.Logger,
	}

	return r, nil
}

func (r *Release) InstallFromTarball(ctx context.Context, app AppConfiguration) error {
	var err error

	var tarballURL string
	{
		r.logger.Debugf(ctx, "getting %#q tarball URL", app.AppName)

		o := func() error {
			tarballURL, err = appcatalog.GetLatestChart(ctx, app.CatalogURL, app.AppName, app.AppVersion)
			if err != nil {
				return microerror.Mask(err)
			}

			return nil
		}

		b := backoff.NewConstant(5*time.Minute, 10*time.Second)
		n := backoff.NewNotifier(r.logger, ctx)

		err = backoff.RetryNotify(o, b, n)
		if err != nil {
			return microerror.Mask(err)
		}

		r.logger.Debugf(ctx, "tarball URL is %#q", tarballURL)
	}

	var tarballPath string
	{
		r.logger.Debugf(ctx, "pulling tarball")

		tarballPath, err = r.helmClient.PullChartTarball(ctx, tarballURL)
		if err != nil {
			return microerror.Mask(err)
		}

		r.logger.Debugf(ctx, "tarball path is %#q", tarballPath)
	}

	var values map[string]interface{}
	{
		err = yaml.Unmarshal([]byte(app.AppValues), &values)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	{
		defer func() {
			fs := afero.NewOsFs()
			err := fs.Remove(tarballPath)
			if err != nil {
				r.logger.Errorf(ctx, err, "deletion of %#q failed", tarballPath)
			}
		}()

		r.logger.Debugf(ctx, "installing %#q", app.AppName)

		opts := helmclient.InstallOptions{
			ReleaseName: app.AppName,
			Wait:        true,
		}
		err = r.helmClient.InstallReleaseFromTarball(ctx,
			tarballPath,
			app.AppNamespace,
			values,
			opts)
		if err != nil {
			return microerror.Mask(err)
		}

		r.logger.Debugf(ctx, "installed %#q", app.AppName)
	}

	return nil
}

func (r *Release) UpdateFromTarball(ctx context.Context, app AppConfiguration) error {
	var err error

	var tarballURL string
	{
		r.logger.Debugf(ctx, "getting %#q tarball URL", app.AppName)

		o := func() error {
			tarballURL, err = appcatalog.GetLatestChart(ctx, app.CatalogURL, app.AppName, app.AppVersion)
			if err != nil {
				return microerror.Mask(err)
			}

			return nil
		}

		b := backoff.NewConstant(5*time.Minute, 10*time.Second)
		n := backoff.NewNotifier(r.logger, ctx)

		err = backoff.RetryNotify(o, b, n)
		if err != nil {
			return microerror.Mask(err)
		}

		r.logger.Debugf(ctx, "tarball URL is %#q", tarballURL)
	}

	var tarballPath string
	{
		r.logger.Debugf(ctx, "pulling tarball")

		tarballPath, err = r.helmClient.PullChartTarball(ctx, tarballURL)
		if err != nil {
			return microerror.Mask(err)
		}

		r.logger.Debugf(ctx, "tarball path is %#q", tarballPath)
	}

	var values map[string]interface{}
	{
		err = yaml.Unmarshal([]byte(app.AppValues), &values)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	{
		defer func() {
			fs := afero.NewOsFs()
			err := fs.Remove(tarballPath)
			if err != nil {
				r.logger.Errorf(ctx, err, "deletion of %#q failed", tarballPath)
			}
		}()

		r.logger.Debugf(ctx, "updating %#q", app.AppName)

		opts := helmclient.UpdateOptions{
			Wait: true,
		}
		err = r.helmClient.UpdateReleaseFromTarball(ctx,
			tarballPath,
			app.AppNamespace,
			app.AppName,
			values,
			opts)
		if err != nil {
			return microerror.Mask(err)
		}

		r.logger.Debugf(ctx, "updated %#q", app.AppName)
	}

	return nil
}

func (r *Release) WaitForAppStatusRelease(ctx context.Context, namespace, appName string, release v1alpha1.AppStatusRelease) error {
	var app v1alpha1.App
	var err error

	o := func() error {
		err = r.k8sClient.CtrlClient().Get(
			ctx,
			types.NamespacedName{Namespace: namespace, Name: appName},
			&app,
		)
		if err != nil {
			return microerror.Mask(err)
		}

		if app.Status.Release.Reason != release.Reason {
			return microerror.Maskf(waitError, "expected '%s', but got '%s'", release.Reason, app.Status.Release.Reason)
		}

		if app.Status.Release.Status != release.Status {
			return microerror.Maskf(waitError, "expected '%s', but got '%s'", release.Status, app.Status.Release.Status)
		}

		return nil
	}

	n := func(err error, t time.Duration) {
		r.logger.Log("level", "debug", "message", fmt.Sprintf("failed to get deleted app '%s': retrying in %s", appName, t), "stack", fmt.Sprintf("%v", err))
	}

	b := backoff.NewExponential(10*time.Minute, 60*time.Second)
	err = backoff.RetryNotify(o, b, n)
	if err != nil {
		return microerror.Mask(err)
	}
	return nil
}

func (r *Release) WaitForDeletedApp(ctx context.Context, namespace, appName string) error {
	var app v1alpha1.App
	var err error

	o := func() error {
		err = r.k8sClient.CtrlClient().Get(
			ctx,
			types.NamespacedName{Namespace: namespace, Name: appName},
			&app,
		)
		if apierrors.IsNotFound(err) {
			// Fall through.
			return nil
		} else if err != nil {
			return microerror.Mask(err)
		}

		return nil
	}

	n := func(err error, t time.Duration) {
		r.logger.Log("level", "debug", "message", fmt.Sprintf("failed to get deleted app '%s': retrying in %s", appName, t), "stack", fmt.Sprintf("%v", err))
	}

	b := backoff.NewExponential(10*time.Minute, 60*time.Second)
	err = backoff.RetryNotify(o, b, n)
	if err != nil {
		return microerror.Mask(err)
	}
	return nil
}

func (r *Release) WaitForDeletedChart(ctx context.Context, namespace, chartName string) error {
	var chart v1alpha1.Chart
	var err error

	o := func() error {
		err = r.k8sClient.CtrlClient().Get(
			ctx,
			types.NamespacedName{Namespace: namespace, Name: chartName},
			&chart,
		)
		if apierrors.IsNotFound(err) {
			// Fall through.
			return nil
		} else if err != nil {
			return microerror.Mask(err)
		}

		return nil
	}

	n := func(err error, t time.Duration) {
		r.logger.Log("level", "debug", "message", fmt.Sprintf("failed to get deleted chart '%s': retrying in %s", chartName, t), "stack", fmt.Sprintf("%v", err))
	}

	b := backoff.NewExponential(10*time.Minute, 60*time.Second)
	err = backoff.RetryNotify(o, b, n)
	if err != nil {
		return microerror.Mask(err)
	}
	return nil
}

func (r *Release) WaitForHelmRelease(ctx context.Context, namespace, hrName string) error {
	var err error

	o := func() error {
		err = r.k8sClient.CtrlClient().Get(
			ctx,
			types.NamespacedName{Namespace: namespace, Name: hrName},
			&helmv2.HelmRelease{},
		)
		if err != nil {
			return microerror.Mask(err)
		}

		return nil
	}

	n := func(err error, t time.Duration) {
		r.logger.Log("level", "debug", "message", fmt.Sprintf("failed to get HelmReleae '%s': retrying in %s", hrName, t), "stack", fmt.Sprintf("%v", err))
	}

	b := backoff.NewExponential(10*time.Minute, 60*time.Second)
	err = backoff.RetryNotify(o, b, n)
	if err != nil {
		return microerror.Mask(err)
	}
	return nil
}

func (r *Release) WaitForDeletedHelmRelease(ctx context.Context, namespace, hrName string) error {
	var err error

	o := func() error {
		err = r.k8sClient.CtrlClient().Get(
			ctx,
			types.NamespacedName{Namespace: namespace, Name: hrName},
			&helmv2.HelmRelease{},
		)
		if apierrors.IsNotFound(err) {
			// Fall through.
			return nil
		} else if err != nil {
			return microerror.Mask(err)
		}

		return nil
	}

	n := func(err error, t time.Duration) {
		r.logger.Log("level", "debug", "message", fmt.Sprintf("failed to get deleted HelmReleae '%s': retrying in %s", hrName, t), "stack", fmt.Sprintf("%v", err))
	}

	b := backoff.NewExponential(10*time.Minute, 60*time.Second)
	err = backoff.RetryNotify(o, b, n)
	if err != nil {
		return microerror.Mask(err)
	}
	return nil
}

func (r *Release) WaitForPod(ctx context.Context, namespace, labelSelector string) error {
	o := func() error {
		pods, err := r.k8sClient.K8sClient().CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{LabelSelector: labelSelector})
		if err != nil {
			return microerror.Mask(err)
		}
		if len(pods.Items) != 1 {
			return microerror.Maskf(waitError, "expected 1 pod but got %d", len(pods.Items))
		}

		pod := pods.Items[0]
		if pod.Status.Phase != corev1.PodRunning {
			return microerror.Maskf(waitError, "expected Pod phase %#q but got %#q", corev1.PodRunning, pod.Status.Phase)
		}

		return nil
	}

	n := func(err error, t time.Duration) {
		r.logger.Log("level", "debug", "message", fmt.Sprintf("failed to get pod with selector '%s': retrying in %s", labelSelector, t), "stack", fmt.Sprintf("%v", err))
	}

	b := backoff.NewExponential(10*time.Minute, 60*time.Second)
	err := backoff.RetryNotify(o, b, n)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func (r *Release) WaitForReleaseStatus(ctx context.Context, namespace, release, status string) error {
	o := func() error {
		rc, err := r.helmClient.GetReleaseContent(ctx, namespace, release)
		if helmclient.IsReleaseNotFound(err) && status == helmclient.StatusUninstalled {
			// Error is expected because we purge releases when deleting.
			return nil
		} else if err != nil {
			return microerror.Mask(err)
		}
		if rc.Status != status {
			return microerror.Maskf(releaseStatusNotMatchingError, "waiting for '%s', current '%s'", status, rc.Status)
		}
		return nil
	}

	n := func(err error, t time.Duration) {
		r.logger.Log("level", "debug", "message", fmt.Sprintf("failed to get release status '%s': retrying in %s", status, t), "stack", fmt.Sprintf("%v", err))
	}

	b := backoff.NewExponential(10*time.Minute, 60*time.Second)
	err := backoff.RetryNotify(o, b, n)
	if err != nil {
		return microerror.Mask(err)
	}
	return nil
}

func (r *Release) WaitForReleaseRevision(ctx context.Context, namespace, release string, revision int) error {
	o := func() error {
		rh, err := r.helmClient.GetReleaseContent(ctx, namespace, release)
		if err != nil {
			return microerror.Mask(err)
		}
		if rh.Revision != revision {
			return microerror.Maskf(releaseVersionNotMatchingError, "waiting for '%d', current '%d'", revision, rh.Revision)
		}
		return nil
	}

	n := func(err error, t time.Duration) {
		r.logger.Log("level", "debug", "message", fmt.Sprintf("failed to get release revision '%d': retrying in %s", revision, t), "stack", fmt.Sprintf("%v", err))
	}

	b := backoff.NewExponential(10*time.Minute, 60*time.Second)
	err := backoff.RetryNotify(o, b, n)
	if err != nil {
		return microerror.Mask(err)
	}
	return nil
}

func (r *Release) WaitForReleaseVersion(ctx context.Context, namespace, release, version string) error {
	o := func() error {
		rh, err := r.helmClient.GetReleaseContent(ctx, namespace, release)
		if err != nil {
			return microerror.Mask(err)
		}
		if rh.Version != version {
			return microerror.Maskf(releaseVersionNotMatchingError, "waiting for '%s', current '%s'", version, rh.Version)
		}
		return nil
	}

	n := func(err error, t time.Duration) {
		r.logger.Log("level", "debug", "message", fmt.Sprintf("failed to get release version '%s': retrying in %s", version, t), "stack", fmt.Sprintf("%v", err))
	}

	b := backoff.NewExponential(10*time.Minute, 60*time.Second)
	err := backoff.RetryNotify(o, b, n)
	if err != nil {
		return microerror.Mask(err)
	}
	return nil
}
