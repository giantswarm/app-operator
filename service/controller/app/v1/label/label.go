package label

// FilterLabels processes the labels set for the app CR. The required labels
// are always set and the ignore labels are removed. This ensures that chart
// CRs and related Kubernetes resources have the correct labels.
func FilterLabels(appLabels, requiredLabels map[string]string, ignoreLabels []string) map[string]string {
	labels := requiredLabels

	labelFilter := map[string]bool{}
	for _, l := range ignoreLabels {
		labelFilter[l] = true
	}

	for k, v := range appLabels {
		//
		if _, ok := labels[k]; !ok {
			labels[k] = v
		}

		if _, ok := labelFilter[k]; !ok {
			labels[k] = v
		}
	}

	return labels
}
