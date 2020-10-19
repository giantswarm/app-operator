// +build k8srequired

package key

func AppCatalogEntryName() string {
	return "giantswarm-prometheus-operator-app-0.4.0"
}

func ControlPlaneTestCatalogStorageURL() string {
	return "https://giantswarm.github.io/control-plane-test-catalog"
}

func DefaultCatalogName() string {
	return "default"
}

func DefaultCatalogStorageURL() string {
	return "https://giantswarm.github.com/default-catalog"
}

func Namespace() string {
	return "giantswarm"
}

func StableCatalogName() string {
	return "giantswarm"
}

func StableCatalogStorageURL() string {
	return "https://giantswarm.github.com/giantswarm-catalog"
}

func TestAppReleaseName() string {
	return "test-app"
}

func UniqueAppVersion() string {
	return "0.0.0"
}
