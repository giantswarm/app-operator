//go:build k8srequired
// +build k8srequired

package key

import (
	"fmt"
	"strings"

	"github.com/giantswarm/app-operator/v6/integration/env"

	"github.com/giantswarm/app-operator/v6/pkg/project"
)

func CatalogConfigMapName() string {
	return "catalog-config"
}

func AppOperatorInTestVersion() string {
	var version string
	if strings.HasSuffix(project.Version(), "-dev") || !env.IsMainBranch() {
		// In case of running the tests against a development version, the artifact is uploaded to the test catalog
		// with the SHA1 postfixed to the version, e.g. app-operator-5.11.0-19b12a1e4e9ea3e9733ae1d3bb6b33830d8c2738.tgz
		version = env.CircleSHA()
	} else {
		// In case of running the tests against a release it is only uploaded to the test catalog with the project version,
		// for example: app-operator-6.0.0.tgz (no SHA1 postfixed version is available)
		version = project.Version()
	}

	return version
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
	return "3.3.0"
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
