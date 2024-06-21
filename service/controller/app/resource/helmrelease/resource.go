package helmrelease

import (
	"time"

	helmv2 "github.com/fluxcd/helm-controller/api/v2beta1"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/app-operator/v6/service/controller/app/controllercontext"
	"github.com/giantswarm/app-operator/v6/service/internal/indexcache"
)

const (
	// Name is the identifier of the resource.
	Name = "helmrelease"

	helmRepositoryKind = "HelmRepository"
)

// Config represents the configuration used to create a new HelmRelease custom resource.
type Config struct {
	// Dependencies.
	IndexCache indexcache.Interface
	Logger     micrologger.Logger
	CtrlClient client.Client

	// Settings.
	WorkloadClusterID            string
	DependencyWaitTimeoutMinutes int
}

// Resource implements the HelmRelease custom resource.
type Resource struct {
	// Dependencies.
	indexCache indexcache.Interface
	logger     micrologger.Logger
	ctrlClient client.Client

	// Settings.
	workloadClusterID            string
	dependencyWaitTimeoutMinutes int
}

// New creates a new configured HelmRelease custom resource.
func New(config Config) (*Resource, error) {
	if config.IndexCache == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.IndexCache$ must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}
	if config.CtrlClient == client.Client(nil) {
		return nil, microerror.Maskf(invalidConfigError, "%T.CtrlClient must not be empty", config)
	}
	if config.DependencyWaitTimeoutMinutes <= 0 {
		return nil, microerror.Maskf(invalidConfigError, "%T.DependencyWaitTimeoutMinutes must be greater than 0", config)
	}

	r := &Resource{
		indexCache: config.IndexCache,
		logger:     config.Logger,
		ctrlClient: config.CtrlClient,

		workloadClusterID:            config.WorkloadClusterID,
		dependencyWaitTimeoutMinutes: config.DependencyWaitTimeoutMinutes,
	}

	return r, nil
}

func (r *Resource) Name() string {
	return Name
}

// addStatusToContext adds the status to the controller context. It will be
// used to set the CR status in the status resource.
func addStatusToContext(cc *controllercontext.Context, reason, status string) {
	cc.Status = controllercontext.Status{
		ChartStatus: controllercontext.ChartStatus{
			Reason: reason,
			Status: status,
		},
	}
}

// configurePause copies pause timestamp from the current resources and resume
// HelmRelease if necessary.
func (r *Resource) configurePause(current, desired *helmv2.HelmRelease) {
	// Get pause timestamp from the current resource. Desired version of the
	// resource carries new timestamp, but we want the old one in case this
	// resource has been previously suspended.
	pauseTs := current.Annotations[annotationHelmReleasePauseStarted]

	// If it is desired to suspend an app, and the current timestamp suggest
	// it is a continuation of the previous act, then we re-use the timestamp.
	if desired.Spec.Suspend && pauseTs != "" {
		// We want to keep the existing pause timestamp.
		desired.Annotations[annotationHelmReleasePauseStarted] = pauseTs
	}

	// Check if pause timestamp has expired.
	if ts, found := desired.Annotations[annotationHelmReleasePauseStarted]; found {
		t, err := time.Parse(time.RFC3339, ts)
		if err != nil {
			// Timestamp invalid, do nothing.
			return
		}

		if time.Since(t) > (time.Minute * time.Duration(r.dependencyWaitTimeoutMinutes)) {
			// Wait timeout is expired, remove pause annotations.
			delete(desired.Annotations, annotationHelmReleasePauseStarted)
			delete(desired.Annotations, annotationHelmReleasePauseReason)
			desired.Spec.Suspend = false
		}
	}
}

// copyHelmRelease creates a new HelmRelease CR object based on the current resource,
// so later we don't need to show unnecessary differences.
func copyHelmRelease(current *helmv2.HelmRelease) *helmv2.HelmRelease {
	newHelmRelease := &helmv2.HelmRelease{
		TypeMeta: metav1.TypeMeta{
			Kind:       helmv2.HelmReleaseKind,
			APIVersion: helmv2.GroupVersion.Group,
		},
	}

	newHelmRelease.Name = current.Name
	newHelmRelease.Namespace = current.Namespace

	newHelmRelease.Annotations = current.Annotations
	newHelmRelease.Labels = current.Labels
	newHelmRelease.Spec = *current.Spec.DeepCopy()

	return newHelmRelease
}

// toHelmRelease converts the input into a HelmRelease.
func toHelmRelease(v interface{}) (*helmv2.HelmRelease, error) {
	if v == nil {
		return &helmv2.HelmRelease{}, nil
	}

	hr, ok := v.(*helmv2.HelmRelease)
	if !ok {
		return nil, microerror.Maskf(wrongTypeError, "expected '%T', got '%T'", &helmv2.HelmRelease{}, v)
	}

	return hr, nil
}
