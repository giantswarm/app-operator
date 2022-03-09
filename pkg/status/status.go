package status

const (
	// ConfigmapMergeFailedStatus is set in the CR status when there is an failure during
	// merge configmaps.
	ConfigmapMergeFailedStatus = "configmap-merge-failed"

	// PendingStatus is set in the CR status when the kubeconfig secret
	// or cluster values configmap does not exist yet.
	PendingStatus = "pending"

	// SecretMergeFailedStatus is set in the CR status when there is an failure during
	// merge secrets.
	SecretMergeFailedStatus = "secret-merge-failed"

	// ValidationFailedStatus is set in the CR status when there is a validation error
	// not related to dependent kubernetes resources.
	ValidationFailedStatus = "validation-failed"
)

var (
	FailedStatus = map[string]bool{
		ConfigmapMergeFailedStatus: true,
		SecretMergeFailedStatus:    true,
	}
)
