package sentrygatherer

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	prometheus "github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/rs/zerolog"
)

type Options struct {
	Token          string
	Organization   string
	APIBaseURL     string
	TickerInterval time.Duration
}

type SentryGatherer struct {
	opts                   Options
	registry               *prometheus.Registry
	metricOrgTeamsTotal    prometheus.Gauge
	metricOrgProjectsTotal prometheus.Gauge
}

func (g *SentryGatherer) Gather() ([]*dto.MetricFamily, error) {
	return g.registry.Gather()
}

type SentryOrganizationProject struct{}
type SentryOrganizationTeam struct{}

func (g *SentryGatherer) gatherOrgProjectsTotal(ctx context.Context) (int, error) {
	client := http.Client{}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/api/0/organizations/%s/projects/", g.opts.APIBaseURL, g.opts.Organization), nil)
	if err != nil {
		return 0, fmt.Errorf("failed to build API URL: %w", err)
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", g.opts.Token))
	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to request data from API: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("unexpected status code %d returned from API", resp.StatusCode)
	}
	defer resp.Body.Close()
	var list []SentryOrganizationProject
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		return 0, fmt.Errorf("failed to decode response from server: %w", err)
	}
	return len(list), nil
}

func (g *SentryGatherer) gatherOrgTeamsTotal(ctx context.Context) (int, error) {
	client := http.Client{}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/api/0/organizations/%s/teams/", g.opts.APIBaseURL, g.opts.Organization), nil)
	if err != nil {
		return 0, fmt.Errorf("failed to build API URL: %w", err)
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", g.opts.Token))
	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to request data from API: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("unexpected status code %d returned from API", resp.StatusCode)
	}
	defer resp.Body.Close()
	var list []SentryOrganizationTeam
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		return 0, fmt.Errorf("failed to decode response from server: %w", err)
	}
	return len(list), nil
}

func (g *SentryGatherer) updateAll(ctx context.Context) {
	logger := zerolog.Ctx(ctx)
	logger.Info().Msg("Updating metrics.")
	count, err := g.gatherOrgProjectsTotal(ctx)
	if err != nil {
		logger.Warn().Err(err).Msg("Failed to gather project totals.")
	}
	g.metricOrgProjectsTotal.Set(float64(count))
	count, err = g.gatherOrgTeamsTotal(ctx)
	if err != nil {
		logger.Warn().Err(err).Msg("Failed to gather team totals.")
	}
	g.metricOrgTeamsTotal.Set(float64(count))
}

func (g *SentryGatherer) Start(ctx context.Context) {
	go func() {
		g.updateAll(ctx)
		ticker := time.NewTicker(g.opts.TickerInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				g.updateAll(ctx)
				continue
			}
		}
	}()
}

func New(opts Options) (*SentryGatherer, error) {
	g := &SentryGatherer{}
	if opts.Token == "" || opts.Organization == "" {
		return nil, fmt.Errorf("Token or Organization not set")
	}
	if opts.APIBaseURL == "" {
		opts.APIBaseURL = "https://sentry.io"
	}
	g.opts = opts
	g.registry = prometheus.NewRegistry()
	g.metricOrgTeamsTotal = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "sentry_organization_teams_total",
		Help: "Number of teams in an organization",
	})
	g.metricOrgProjectsTotal = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "sentry_organization_projects_total",
		Help: "Number of projects in an organization",
	})
	if err := g.registry.Register(g.metricOrgTeamsTotal); err != nil {
		return g, err
	}
	if err := g.registry.Register(g.metricOrgProjectsTotal); err != nil {
		return g, err
	}
	return g, nil
}
