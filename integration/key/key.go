// +build k8srequired

package key

import "fmt"

func AppCatalogConfigMapName() string {
	return "appcatalog-config"
}

func AppCatalogEntryName() string {
	return "giantswarm-prometheus-operator-app-0.3.4"
}

func ChartOperatorName() string {
	return "chart-operator"
}

func ChartOperatorUniqueName() string {
	return fmt.Sprintf("%s-unique", ChartOperatorName())
}

func ChartOperatorVersion() string {
	return "2.7.0"
}

func ControlPlaneTestCatalogStorageURL() string {
	return "https://giantswarm.github.io/control-plane-test-catalog"
}

func DefaultCatalogName() string {
	return "default"
}

func DefaultCatalogStorageURL() string {
	return "https://giantswarm.github.io/default-catalog"
}

func Namespace() string {
	return "giantswarm"
}

func StableCatalogName() string {
	return "giantswarm"
}

func TestAppName() string {
	return "test-app"
}

func UniqueAppVersion() string {
	return "0.0.0"
}

func UserConfigMapName() string {
	return "user-config"
}
