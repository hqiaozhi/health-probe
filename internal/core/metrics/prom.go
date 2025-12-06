package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

type Metrics struct {
	// http
	HttpDuration   *prometheus.HistogramVec
	HttpStatusCode *prometheus.GaugeVec
	HttpSuccess    *prometheus.GaugeVec
	// tcp
	TcpDuration *prometheus.HistogramVec
	TcpSuccess  *prometheus.GaugeVec
	// dns
	DnsDuration *prometheus.HistogramVec
	DnsSuccess  *prometheus.GaugeVec
	// ICMP
	IcmpDuration *prometheus.HistogramVec
	IcmpSuccess  *prometheus.GaugeVec
}

func New() *Metrics {
	return &Metrics{
		HttpDuration:   HttpProbeDuration,
		HttpStatusCode: HttpProbeStatusCode,
		HttpSuccess:    HttpProbeSuccess,
		TcpDuration:    TcpProbeDuration,
		TcpSuccess:     TcpProbeSuccess,
		DnsDuration:    DnsProbeDuration,
		DnsSuccess:     DnsProbeSuccess,
		IcmpDuration:   IcmpProbeDuration,
		IcmpSuccess:    IcmpProbeSuccess,
	}
}
