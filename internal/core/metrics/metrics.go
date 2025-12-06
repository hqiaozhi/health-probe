package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	// HTTP 探测指标
	HttpProbeDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_probe_duration_seconds",
			Help:    "HTTP 请求的持续时间（秒）",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"target"},
	)

	HttpProbeStatusCode = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "http_probe_status_code",
			Help: "HTTP 请求的响应状态码（0 表示请求失败）",
		},
		[]string{"target"},
	)

	HttpProbeSuccess = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "http_probe_success",
			Help: "HTTP 请求是否成功（1=成功，0=失败）",
		},
		[]string{"target"},
	)

	// TCP 探测指标
	TcpProbeDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "tcp_probe_duration_seconds",
			Help:    "TCP 探测的持续时间（秒）",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"target"},
	)

	TcpProbeSuccess = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "tcp_probe_success",
			Help: "TCP 探测是否成功（1=成功，0=失败）",
		},
		[]string{"target"},
	)

	// DNS 探测指标
	DnsProbeDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "dns_probe_duration_seconds",
			Help:    "DNS 请求的响应时间（秒）",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"target"},
	)

	DnsProbeSuccess = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "dns_probe_success",
			Help: "DNS 请求是否成功（1=成功，0=失败）",
		},
		[]string{"target"},
	)
	// ICMP 探测指标
	IcmpProbeDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "icmp_probe_duration_seconds",
			Help:    "ICMP 探测的持续时间（秒）",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"target"},
	)

	IcmpProbeSuccess = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "icmp_probe_success",
			Help: "ICMP 探测是否成功（1=成功，0=失败）",
		},
		[]string{"target"},
	)
)

// 注册所有指标
func init() {
	prometheus.MustRegister(HttpProbeDuration)
	prometheus.MustRegister(HttpProbeStatusCode)
	prometheus.MustRegister(HttpProbeSuccess)
	prometheus.MustRegister(TcpProbeDuration)
	prometheus.MustRegister(TcpProbeSuccess)
	prometheus.MustRegister(DnsProbeDuration)
	prometheus.MustRegister(DnsProbeSuccess)
	prometheus.MustRegister(IcmpProbeDuration)
	prometheus.MustRegister(IcmpProbeSuccess)
}
