package prometheus

import (
	"github.com/prometheus/client_golang/prometheus"
)

const (
	prefix = "aws_subnet_exporter_"
)

var (
	labels = []string{"vpcid", "subnetid", "cidrblock", "az", "name"}

	// Prometheus gauge vector for available IPs in subnets
	AvailableIPs = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: prefix + "available_ips",
		Help: "Available IPs in subnets",
	}, labels)

	// Prometheus gauge vector for max IPs in subnets
	MaxIPs = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: prefix + "max_ips",
		Help: "Max host IPs in subnet",
	}, labels)

	// Prometheus gauge vector for used prefixes in subnets
	UsedPrefixes = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: prefix + "used_prefixes",
		Help: "Used prefixes in subnets",
	}, labels)

	// Prometheus gauge vector for available prefixes in subnets
	AvailablePrefixes = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: prefix + "available_prefixes",
		Help: "Available prefixes in subnets",
	}, labels)

)

// Prometheus register metrics
func RegisterMetrics() {
	prometheus.MustRegister(AvailableIPs)
	prometheus.MustRegister(MaxIPs)
	prometheus.MustRegister(UsedPrefixes)
	prometheus.MustRegister(AvailablePrefixes)
}
