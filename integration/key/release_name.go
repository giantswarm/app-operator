// +build k8srequired

package key

func AppOperatorReleaseName() string {
	return "app-operator"
}

func CustomResourceReleaseName() string {
	return "apiextensions-app-e2e-chart"
}

func TestAppReleaseName() string {
	return "test-app"
}

func TestAppCatalogReleaseName() string {
	return "test-app-catalog"
}
