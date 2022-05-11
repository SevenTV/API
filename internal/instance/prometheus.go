package instance

import "github.com/prometheus/client_golang/prometheus"

type Prometheus interface {
	Register(r prometheus.Registerer)
}
