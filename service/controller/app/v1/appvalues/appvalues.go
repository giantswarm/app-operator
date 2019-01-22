package appvalues

import "github.com/giantswarm/microerror"

// MergeValues merges configmap or secret values to produce a single set of
// values. app and appCatalog values are unioned.
func MergeValues(appValues, appCatalogValues map[string]interface{}) (map[string]interface{}, error) {
	var err error
	values := make(map[string]interface{})

	values, err = union(appValues, appCatalogValues)
	if err != nil {
		return values, microerror.Mask(err)
	}

	return values, nil
}

func union(a, b map[string]interface{}) (map[string]interface{}, error) {
	if a == nil {
		return b, nil
	}

	for k, v := range b {
		_, ok := a[k]
		if ok {
			// The values have at least one shared key. We cannot decide which
			// value should be applied.
			return nil, microerror.Maskf(invalidExecutionError, "values share the same key %#q", k)
		}
		a[k] = v
	}
	return a, nil
}
