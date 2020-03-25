// +build k8srequired

package key

func AppOperatorName() string {
	return "app-operator"
}

// AppOperatorChartName returns the name of the appr chart.
// TODO Remove once the operator is flattened.
//
// https://github.com/giantswarm/giantswarm/issues/7895
//
func AppOperatorChartName() string {
	return "app-operator-chart"
}

func AppOperatorVersion() string {
	return "1.0.0"
}

func ChartOperatorName() string {
	return "chart-operator"
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
