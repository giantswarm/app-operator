// +build k8srequired

package key

func AppOperatorVersion() string {
	return "1.0.0"
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

func TestAppReleaseName() string {
	return "test-app"
}
