package appcatalogentry

import "github.com/prometheus/client_golang/prometheus"

const (
	PrometheusNamespace = "app_operator"
	PrometheusSubsystem = Name
)

var (
	histogram = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: PrometheusNamespace,
			Subsystem: PrometheusSubsystem,
			Name:      "event",
			Help:      "Histogram for events within the appcatalogentry resource.",
		},
		[]string{"event"},
	)
)

func init() {
	prometheus.MustRegister(histogram)
}
