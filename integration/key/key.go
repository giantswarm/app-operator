//go:build k8srequired
// +build k8srequired

package key

import (
	"fmt"

	"github.com/giantswarm/app-operator/v5/pkg/project"
)

func CatalogConfigMapName() string {
	return "catalog-config"
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
	return "2.24.0"
}

func ControlPlaneCatalogName() string {
	return "control-plane-catalog"
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

func GiantSwarmNamespace() string {
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
	return "test-workload"
}
