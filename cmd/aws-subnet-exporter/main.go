package main

import (
	"flag"
	"net/http"
	"time"

	"github.com/ministryofjustice/aws-subnet-exporter/pkg/aws"
	prom "github.com/ministryofjustice/aws-subnet-exporter/pkg/prometheus"
	"github.com/ministryofjustice/aws-subnet-exporter/pkg/utils"
	log "github.com/sirupsen/logrus"
)

const (
	errGoRoutineStopped = "go routine for getting subnets stopped"
	metricsEndpoint     = "/metrics"
	healthEndpoint      = "/healthz"
)

var (
	port   = flag.String("port", "8080", "The port to listen on for HTTP requests.")
	region = flag.String("region", "eu-west-2", "AWS region")
	filter = flag.String("filter", "*", "Filter subnets by tag regex when calling AWS (assumes tag key is Name")
	period = flag.Duration("period", 60*time.Second, "Period for calling AWS in seconds")
	debug  = flag.Bool("debug", false, "Enable debug logging")
)

func init() {
	flag.Parse()
	utils.SetupLogger(debug)
	prom.RegisterMetrics()
}

func main() {
	log.WithFields(log.Fields{"port": *port, "region": *region, "filter": *filter, "period": *period, "endpoint": metricsEndpoint}).Info("Starting aws-subnet-exporter")
	client, err := aws.InitEC2Client(*region)
	if err != nil {
		log.Fatal(err)
	}

	cancel := make(chan struct{})

	ticker := time.NewTicker(*period)
	defer ticker.Stop()

	go func() {
		for {
			subnets, err := aws.GetSubnets(client, *filter)
			if err != nil {
				log.Fatal(err)
			}
			for _, v := range subnets {
				prom.AvailableIPs.WithLabelValues(v.VPCID, v.SubnetID, v.CIDRBlock, v.AZ, v.Name).Set(v.AvailableIPs)
				prom.MaxIPs.WithLabelValues(v.VPCID, v.SubnetID, v.CIDRBlock, v.AZ, v.Name).Set(v.MaxIPs)
				prom.UsedPrefixes.WithLabelValues(v.VPCID, v.SubnetID, v.CIDRBlock, v.AZ, v.Name).Set(float64(v.UsedPrefixes))
				prom.AvailablePrefixes.WithLabelValues(v.VPCID, v.SubnetID, v.CIDRBlock, v.AZ, v.Name).Set(float64(len(v.AvailablePrefixes)))
			}

			select {
			case <-ticker.C:
				continue
			case <-cancel:
				log.Fatal(errGoRoutineStopped)
			}
		}
	}()

	log.WithFields(log.Fields{"endpoint": metricsEndpoint, "port": port}).Info("Starting metrics web server")
	http.Handle(metricsEndpoint, prom.Handler)
	http.Handle(healthEndpoint, http.HandlerFunc(utils.HealthHandler))
	log.Fatal(http.ListenAndServe(":"+*port, nil))
}
