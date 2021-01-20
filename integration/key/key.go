// +build k8srequired

package key

import (
	"fmt"

	"github.com/giantswarm/app-operator/v3/pkg/project"
)

func AppCatalogConfigMapName() string {
	return "appcatalog-config"
}

func AppCatalogEntryName() string {
	return "giantswarm-prometheus-operator-app-0.3.4"
}

func AppOperatorUniqueName() string {
	return fmt.Sprintf("%s-unique", project.Name())
}

func ChartOperatorName() string {
	return "chart-operator"
}

func ChartOperatorUniqueName() string {
	return fmt.Sprintf("%s-unique", ChartOperatorName())
}

func ChartOperatorVersion() string {
	return "2.5.1"
}

func ControlPlaneTestCatalogName() string {
	return "control-plane-test-catalog"
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

func StableCatalogStorageURL() string {
	return "https://giantswarm.github.io/giantswarm-catalog"
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

func WorkloadClusterNamespace() string {
	return "workload-test"
}
