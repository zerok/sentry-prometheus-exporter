package main

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
	"github.com/spf13/pflag"
	"github.com/zerok/sentry-prometheus-exporter/internal/sentrygatherer"
)

func main() {
	logger := zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr}).Level(zerolog.InfoLevel).With().Timestamp().Logger()
	var addr string
	var orgName string
	var tickerInterval time.Duration
	var verbose bool
	pflag.StringVar(&addr, "addr", "localhost:9200", "Address to listen on")
	pflag.StringVar(&orgName, "organization", "", "Name of the organization to watch")
	pflag.DurationVar(&tickerInterval, "ticker-interval", time.Second*60, "Update ticker interval")
	pflag.BoolVar(&verbose, "verbose", false, "Verbose logging")
	pflag.Parse()

	if verbose {
		logger = logger.Level(zerolog.DebugLevel)
	}

	ctx := context.Background()

	g, err := sentrygatherer.New(sentrygatherer.Options{
		Token:          os.Getenv("SENTRY_TOKEN"),
		Organization:   orgName,
		TickerInterval: tickerInterval,
	})
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to setup gatherer.")
	}
	g.Start(logger.WithContext(ctx))
	srv := &http.Server{}
	srv.Handler = promhttp.HandlerFor(g, promhttp.HandlerOpts{
		EnableOpenMetrics: true,
	})
	srv.Addr = addr
	logger.Info().Msgf("Starting server on %s", addr)
	if err := srv.ListenAndServe(); err != nil {
		logger.Fatal().Err(err).Msg("Failed to start HTTP server.")
	}
}
