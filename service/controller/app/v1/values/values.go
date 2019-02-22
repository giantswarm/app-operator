package values

import (
	"bytes"

	"github.com/giantswarm/microerror"
	yaml "gopkg.in/yaml.v2"
)

// MergeConfigMapData is a wrapper for MergeData that accepts and returns
// string maps. This is needed because configmaps use string maps but secrets
// use byte array maps. All keys in the string maps should contain YAML.
func MergeConfigMapData(appConfigMap, catalogConfigMap map[string]string) (map[string]string, error) {
	appData := toByteArrayMap(appConfigMap)
	catalogData := toByteArrayMap(catalogConfigMap)

	mergedData, err := MergeData(appData, catalogData)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	result := toStringMap(mergedData)

	return result, nil
}

// MergeData merges app and catalog config. It accepts byte array maps of YAML
// data. If both maps contain the same key then the YAML is merged but a failed
// execution error is returned if the YAML contains the same top level keys.
//
// TODO: Fix bug with quotes in YAML being stripped by conversion to and from
// map[string]interface{} types.
//
// TODO: Perform deep merge of YAML values rather than current basic merge
// provided by the union function.
//
func MergeData(appData, catalogData map[string][]byte) (map[string][]byte, error) {
	result := make(map[string][]byte)

	for appKey, appYaml := range appData {
		catalogYaml, ok := catalogData[appKey]
		if ok {
			// Merge YAML as both app and catalog maps contain the same key.
			mergedYaml, err := mergeYaml(appYaml, catalogYaml)
			if err != nil {
				return nil, microerror.Mask(err)
			}

			result[appKey] = mergedYaml
		} else {
			// Key only appears in the app level map.
			result[appKey] = appYaml
		}
	}

	for catalogKey, catalogYaml := range catalogData {
		_, ok := result[catalogKey]
		if !ok {
			// Add any keys that are only in the catalog level map.
			result[catalogKey] = catalogYaml
		}
	}

	return result, nil
}

func mergeYaml(appYaml, catalogYaml []byte) ([]byte, error) {
	var err error

	// Parse app YAML into a map of values.
	appValues := make(map[string]interface{})
	err = yaml.Unmarshal(appYaml, appValues)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	// Parse catalog YAML into a map of values.
	catalogValues := make(map[string]interface{})
	err = yaml.Unmarshal(catalogYaml, catalogValues)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	// Do basic merge of top level keys in both maps.
	resultValues, err := union(appValues, catalogValues)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	// Convert merged values back to YAML.
	resultYaml, err := yaml.Marshal(resultValues)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	// Trim leading and trailing whitespace.
	result := bytes.Trim(resultYaml, "\n")

	return result, nil
}

func toByteArrayMap(input map[string]string) map[string][]byte {
	result := make(map[string][]byte)

	for k, v := range input {
		result[k] = []byte(v)
	}

	return result
}

func toStringMap(input map[string][]byte) map[string]string {
	result := make(map[string]string)

	for k, v := range input {
		result[k] = string(v)
	}

	return result
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
			return nil, microerror.Maskf(failedExecutionError, "values share identical key %#q", k)
		}
		a[k] = v
	}
	return a, nil
}
