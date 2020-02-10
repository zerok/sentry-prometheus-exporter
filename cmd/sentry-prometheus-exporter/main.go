package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/pflag"
	"github.com/zerok/sentry-prometheus-exporter/internal/sentrygatherer"
)

func main() {
	var addr string
	var orgName string
	var tickerInterval time.Duration
	pflag.StringVar(&addr, "addr", "localhost:9200", "Address to listen on")
	pflag.StringVar(&orgName, "organization", "", "Name of the organization to watch")
	pflag.DurationVar(&tickerInterval, "ticker-interval", time.Second*60, "Update ticker interval")
	pflag.Parse()

	ctx := context.Background()

	g, err := sentrygatherer.New(sentrygatherer.Options{
		Token:          os.Getenv("SENTRY_TOKEN"),
		Organization:   orgName,
		TickerInterval: tickerInterval,
	})
	if err != nil {
		log.Fatal(err.Error())
	}
	g.Start(ctx)
	srv := &http.Server{}
	srv.Handler = promhttp.HandlerFor(g, promhttp.HandlerOpts{
		EnableOpenMetrics: true,
	})
	srv.Addr = addr
	if err := srv.ListenAndServe(); err != nil {
		log.Fatal(err.Error())
	}
}
