package values

import (
	"github.com/giantswarm/microerror"
	yaml "gopkg.in/yaml.v2"
)

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

func MergeData(appData, catalogData map[string][]byte) (map[string][]byte, error) {
	result := make(map[string][]byte)

	for appKey, appYaml := range appData {
		catalogYaml, ok := catalogData[appKey]
		if ok {
			mergedYaml, err := mergeYaml(appYaml, catalogYaml)
			if err != nil {
				return nil, microerror.Mask(err)
			}

			result[appKey] = mergedYaml
		} else {
			result[appKey] = appYaml
		}
	}

	for catalogKey, catalogYaml := range catalogData {
		_, ok := result[catalogKey]
		if !ok {
			result[catalogKey] = catalogYaml
		}
	}

	return result, nil
}

func mergeYaml(appYaml, catalogYaml []byte) ([]byte, error) {
	var err error

	appValues := make(map[string]interface{})
	catalogValues := make(map[string]interface{})

	err = yaml.Unmarshal(appYaml, appValues)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	err = yaml.Unmarshal(catalogYaml, catalogValues)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	resultValues, err := union(appValues, catalogValues)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	resultYaml, err := yaml.Marshal(resultValues)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return resultYaml, nil
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
