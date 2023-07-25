package monitoring

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog/log"
	"net/http"
)

func Setup() *Metrics {

	reg := prometheus.NewRegistry()
	m := NewMetrics(reg)

	return m
}

func Run(reg *prometheus.Registry) {
	promHandler := promhttp.HandlerFor(reg, promhttp.HandlerOpts{})

	http.Handle("/metrics", promHandler)
	log.Panic().Err(http.ListenAndServe(":8081", nil)).Msg("Metrics endpoint failed")
}

type Metrics struct {
	RequestDuration *prometheus.HistogramVec
	Registry        *prometheus.Registry
}

func NewMetrics(reg *prometheus.Registry) *Metrics {
	m := &Metrics{
		Registry: reg,
		RequestDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: "fhir_bomber",
			Name:      "request_duration_milliseconds",
			Help:      "Requests duration in milliseconds",
		},
			[]string{
				"name",
				"code",
			},
		),
	}
	reg.MustRegister(m.RequestDuration)
	return m
}
