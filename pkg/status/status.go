package status

import (
	helmv2 "github.com/fluxcd/helm-controller/api/v2beta1"
	fluxmeta "github.com/fluxcd/pkg/apis/meta"
	sourcev1 "github.com/fluxcd/source-controller/api/v1beta2"
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
func GetDesiredStatus(hrStatus helmv2.HelmReleaseStatus, hcStatus sourcev1.HelmChartStatus) v1alpha1.AppStatus {
	// We are primarily interested in the Helm release status, not the HelmRelease CR status
	// that only represents the former, therefore we first look for the "Released" condition type.
	condition := apimeta.FindStatusCondition(hrStatus.Conditions, helmv2.ReleasedCondition)

	if condition != nil {
		// We start with successful reasons for these indicate desired state we want
		// the release to be in. We try to map these statuses into known App Platform
		// statuses.
		s, ok := releasedReasonSuccessMapping[condition.Reason]
		if ok {
			return v1alpha1.AppStatus{
				AppVersion: hrStatus.LastAppliedRevision,
				Release: v1alpha1.AppStatusRelease{
					LastDeployed: condition.LastTransitionTime,
					Reason:       condition.Message,
					Status:       s,
				},
				Version: hrStatus.LastAppliedRevision,
			}
		}

		// We continue with failed reasons for these indicate Helm release end up
		// in a bad state. We try to identify some of the known errors.
		s, ok = releasedReasonFailMapping[condition.Reason]
		if ok {
			s = lookForKnownStatus(s, condition.Message)

			return v1alpha1.AppStatus{
				AppVersion: hrStatus.LastAppliedRevision,
				Release: v1alpha1.AppStatusRelease{
					LastDeployed: condition.LastTransitionTime,
					Reason:       condition.Message,
					Status:       s,
				},
				Version: hrStatus.LastAppliedRevision,
			}
		}

		// Otherwise, if no mapping is found, we return an unknown status.
		return v1alpha1.AppStatus{
			AppVersion: hrStatus.LastAppliedRevision,
			Release: v1alpha1.AppStatusRelease{
				LastDeployed: condition.LastTransitionTime,
				Reason:       condition.Message,
				Status:       helmclient.StatusUnknown,
			},
			Version: hrStatus.LastAppliedRevision,
		}
	}

	// If no "Released" condition type is found we are then interested in the "Ready"
	// condition type that indicates the status of the last reconciliation request, which
	// we try to use to infer information from. This condition does not necessarily tell us
	// the Helm release status, but can tell us what got in the way of successfully reconciling
	// HelmRelease CR.
	condition = apimeta.FindStatusCondition(hrStatus.Conditions, fluxmeta.ReadyCondition)
	if condition != nil {
		// We again try to map this condition reason to a known types to App Platform.
		s, ok := readyReasonMapping[condition.Reason]
		if ok {
			return v1alpha1.AppStatus{
				AppVersion: hrStatus.LastAppliedRevision,
				Release: v1alpha1.AppStatusRelease{
					LastDeployed: condition.LastTransitionTime,
					Reason:       condition.Message,
					Status:       s,
				},
				Version: hrStatus.LastAppliedRevision,
			}
		}

		// The ArtifactFailedReason does not provide any granularity unfortunately.
		// In general it informs of problem with the artifact, but does not say anything
		// about its nature, hence we can't infer from it wheather it is a pulling
		// problem, missing version, missing chart, or something else. Worse so, it also
		// plays a role of transient state used by Helm Controller whenever it sees artifact
		// is not ready what has nothing to do with any probmes, but with the fact HelmChart
		// CR is being reconciled.
		// Because of that, this reason is not really good for us and breaks the repository
		// failover mechanism, by tricking App Operator falsely switch repositories sometimes.
		// Hence this reason needs special attention.
		if condition.Reason == helmv2.ArtifactFailedReason {
			if apimeta.IsStatusConditionTrue(hcStatus.Conditions, sourcev1.FetchFailedCondition) {
				return v1alpha1.AppStatus{
					AppVersion: hrStatus.LastAppliedRevision,
					Release: v1alpha1.AppStatusRelease{
						LastDeployed: condition.LastTransitionTime,
						Reason:       condition.Message,
						Status:       ChartPullFailedStatus,
					},
					Version: hrStatus.LastAppliedRevision,
				}
			}

			if apimeta.IsStatusConditionTrue(hcStatus.Conditions, sourcev1.StorageOperationFailedCondition) {
				return v1alpha1.AppStatus{
					AppVersion: hrStatus.LastAppliedRevision,
					Release: v1alpha1.AppStatusRelease{
						LastDeployed: condition.LastTransitionTime,
						Reason:       condition.Message,
						Status:       ChartPullFailedStatus,
					},
					Version: hrStatus.LastAppliedRevision,
				}
			}

			return v1alpha1.AppStatus{
				AppVersion: hrStatus.LastAppliedRevision,
				Release: v1alpha1.AppStatusRelease{
					LastDeployed: condition.LastTransitionTime,
					Reason:       condition.Message,
					Status:       PendingStatus,
				},
				Version: hrStatus.LastAppliedRevision,
			}
		}

		return v1alpha1.AppStatus{
			AppVersion: hrStatus.LastAppliedRevision,
			Release: v1alpha1.AppStatusRelease{
				LastDeployed: condition.LastTransitionTime,
				Reason:       condition.Message,
				Status:       helmclient.StatusUnknown,
			},
			Version: hrStatus.LastAppliedRevision,
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
