package status

import (
	helmv2 "github.com/fluxcd/helm-controller/api/v2beta1"
	fluxmeta "github.com/fluxcd/pkg/apis/meta"
	"github.com/giantswarm/apiextensions-application/api/v1alpha1"
	"github.com/giantswarm/helmclient/v4/pkg/helmclient"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
)

// GetDesiredStatus decides the desired status for App CR based on the
// HelmRelease CR conditions. We are primarily interested in `Released`
// condition, for it tell which state the Helm release is in, so we look
// for this first. We then try to map it into known App Platform statuses.
// If no `Released` condition is found, we then look for `Ready` condition,
// for it informs of general problems encountered on reconciliation.
func GetDesiredStatus(status helmv2.HelmReleaseStatus) v1alpha1.AppStatus {
	// We are primarily interested in the Helm release status, not the HelmRelease CR status
	// that only represents the former, therefore we first look for the "Released" condition type.
	condition := apimeta.FindStatusCondition(status.Conditions, helmv2.ReleasedCondition)

	if condition != nil {
		// We start with successful reasons for these indicate desired state we want
		// the release to be in. We try to map these statuses into known App Platform
		// statuses.
		s, ok := releasedReasonSuccessMapping[condition.Reason]
		if ok {
			return v1alpha1.AppStatus{
				AppVersion: status.LastAppliedRevision,
				Release: v1alpha1.AppStatusRelease{
					LastDeployed: condition.LastTransitionTime,
					Reason:       condition.Message,
					Status:       s,
				},
				Version: status.LastAppliedRevision,
			}
		}

		// We continue with failed reasons for these indicate Helm release end up
		// in a bad state. We try to identify some of the known errors.
		s, ok = releasedReasonFailMapping[condition.Reason]
		if ok {
			s = lookForKnownStatus(s, condition.Message)

			return v1alpha1.AppStatus{
				AppVersion: status.LastAppliedRevision,
				Release: v1alpha1.AppStatusRelease{
					LastDeployed: condition.LastTransitionTime,
					Reason:       condition.Message,
					Status:       s,
				},
				Version: status.LastAppliedRevision,
			}
		}

		// Otherwise, if no mapping is found, we return an unknown status.
		return v1alpha1.AppStatus{
			AppVersion: status.LastAppliedRevision,
			Release: v1alpha1.AppStatusRelease{
				LastDeployed: condition.LastTransitionTime,
				Reason:       condition.Message,
				Status:       helmclient.StatusUnknown,
			},
			Version: status.LastAppliedRevision,
		}
	}

	// If no "Released" condition type is found we are then interested in the "Ready"
	// condition type that indicates the status of the last reconciliation request, which
	// we try to use to infer information from. This condition does not necessarily tell us
	// the Helm release status, but can tell us what got in the way of successfully reconciling
	// HelmRelease CR.
	condition = apimeta.FindStatusCondition(status.Conditions, fluxmeta.ReadyCondition)
	if condition != nil {
		// We again try to map this condition reason to a known types to App Platform.
		s, ok := readyReasonMapping[condition.Reason]
		if ok {
			return v1alpha1.AppStatus{
				AppVersion: status.LastAppliedRevision,
				Release: v1alpha1.AppStatusRelease{
					LastDeployed: condition.LastTransitionTime,
					Reason:       condition.Message,
					Status:       s,
				},
				Version: status.LastAppliedRevision,
			}
		}

		return v1alpha1.AppStatus{
			AppVersion: status.LastAppliedRevision,
			Release: v1alpha1.AppStatusRelease{
				LastDeployed: condition.LastTransitionTime,
				Reason:       condition.Message,
				Status:       helmclient.StatusUnknown,
			},
			Version: status.LastAppliedRevision,
		}
	}

	return v1alpha1.AppStatus{}
}

// lookForKnownStatus gets executed on spotting known, faulty, "Released" type
// condition and tries to match the error message to one of the known erros the
// App Platform has been so far informing users of.
func lookForKnownStatus(status, message string) string {
	known := map[string]func(string) bool{
		ValuesSchemaViolation:     isHelmSchemaValidation,
		ValidationFailedStatus:    isValidationFailedErrorMsg,
		InvalidManifestStatus:     isInvalidManifest,
		AlreadyExistsStatus:       isResourceAlreadyExists,
		ReleaseNotInstalledStatus: isReleaseNameInvalid,
	}

	for s, f := range known {
		if f(message) {
			return s
		}
	}

	return status
}
