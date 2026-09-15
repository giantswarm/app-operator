//go:build k8srequired
// +build k8srequired

package key

import (
	"fmt"
	"strings"

	"github.com/giantswarm/app-operator/v7/integration/env"

	"github.com/giantswarm/app-operator/v7/pkg/project"
)

func CatalogConfigMapName() string {
	return "catalog-config"
}

func AppOperatorInTestVersion() string {
	var version string
	if strings.HasSuffix(project.Version(), "-dev") || !env.IsMainBranch() {
		// In case of running the tests against a development version, the artifact is uploaded to the test catalog
		// with the r[CRC32_branch_name]t[YYYYMMDD][HHMMSS]h[commit_SHA] version.
		version = env.BuildVersion()
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

func TestAppVersion() string {
	return "1.0.0"
}

func TestAppTarballUrl() string {
	return fmt.Sprintf("%s/%s-%s.tgz", DefaultCatalogStorageURL(), TestAppName(), TestAppVersion())
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
