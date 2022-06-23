package prometheus

import (
	"github.com/prometheus/client_golang/prometheus"
)

type Instance interface {
	Register(r prometheus.Registerer)
}

type Options struct {
	Labels prometheus.Labels
}

func New(o Options) Instance {
	return &promInst{}
}

type promInst struct {
}

func (m *promInst) Register(r prometheus.Registerer) {
}
