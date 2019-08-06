package chartoperator

// Values represents the values to be passed to Helm commands related to
// chart-operator chart.
type Values struct {
	ClusterDNSIP string `json:"clusterDNSIP"`
	Image        Image  `json:"image"`
	Tiller       Tiller `json:"tiller"`
}

// Image holds the image settings for chart-operator chart.
type Image struct {
	Registry string `json:"registry"`
}

// Tiller holds the Tiller settings for chart-operator chart.
type Tiller struct {
	Namespace string `json:"namespace"`
}
