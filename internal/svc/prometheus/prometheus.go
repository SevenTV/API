package prometheus

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/seventv/api/internal/instance"
)

type Options struct {
	Labels prometheus.Labels
}

func New(o Options) instance.Prometheus {
	return &Instance{}
}

type Instance struct {
}

func (m *Instance) Register(r prometheus.Registerer) {
}
