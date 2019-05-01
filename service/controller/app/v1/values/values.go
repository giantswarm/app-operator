package values

import (
	"github.com/giantswarm/helmclient"
	"github.com/giantswarm/microerror"
	yaml "gopkg.in/yaml.v2"
)

// MergeConfigMapData merges configmap data into a single block of YAML that
// is stored in the configmap associated with the relevant chart CR.
func MergeConfigMapData(destMap, srcMap map[string]string) (map[string]string, error) {
	result, err := mergeData(toByteSliceMap(destMap), toByteSliceMap(srcMap))
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return toStringMap(result), nil
}

// MergeSecretData merges secret data into a single block of YAML that
// is stored in the secret associated with the relevant chart CR.
func MergeSecretData(destMap, srcMap map[string][]byte) (map[string][]byte, error) {
	result, err := mergeData(destMap, srcMap)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return result, nil
}

// mergeData contains the shared logic that is common to merging configmap and
// secret data.
func mergeData(destMap, srcMap map[string][]byte) (map[string][]byte, error) {
	mergedData, err := helmclient.MergeValues(destMap, srcMap)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	bytes, err := yaml.Marshal(mergedData)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	// Key name is just for display purposes. The only restriction is the
	// configmap or secret storing data for the chart CR can only have a single
	// key with YAML values.
	result := map[string][]byte{
		"values": bytes,
	}

	return result, nil
}

// toByteSliceMap converts from a string map to a byte slice map.
func toByteSliceMap(input map[string]string) map[string][]byte {
	result := map[string][]byte{}

	for k, v := range input {
		result[k] = []byte(v)
	}

	return result
}

// toStringMap converts from a byte slice map to a string map.
func toStringMap(input map[string][]byte) map[string]string {
	result := map[string]string{}

	for k, v := range input {
		result[k] = string(v)
	}

	return result
}
