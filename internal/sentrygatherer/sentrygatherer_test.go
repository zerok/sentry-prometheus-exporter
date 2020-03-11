package sentrygatherer

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	dto "github.com/prometheus/client_model/go"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

func TestCoreMetrics(t *testing.T) {
	var g *SentryGatherer
	var err error
	// If no token and no organisation name is provided, an error should be
	// returned:
	_, err = New(Options{})
	require.Error(t, err)

	_, err = New(Options{
		Token: "...",
	})
	require.Error(t, err)

	_, err = New(Options{
		Organization: "...",
	})
	require.Error(t, err)

	g, err = New(Options{
		Organization: "...",
		Token:        "...",
	})
	require.NoError(t, err)
	metrics, err := g.Gather()
	require.NoError(t, err)
	require.NotNil(t, metrics)

	requireMetric(t, metrics, "sentry_organization_teams_total")
	requireMetric(t, metrics, "sentry_organization_projects_total")
}

func TestNumberOfProjects(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/api/0/organizations/exists/projects/", func(w http.ResponseWriter, r *http.Request) {
		sendJSONFile(t, w, "projects.json")
	})
	srv := httptest.NewServer(handler)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/0/organizations/exists/projects/")
	require.NoError(t, err)
	require.Equal(t, 200, resp.StatusCode)

	g, err := New(Options{
		Token:        "...",
		Organization: "exists",
		APIBaseURL:   srv.URL,
	})
	require.NoError(t, err)
	require.NotNil(t, g)

	g.updateAll(context.Background())
	metrics, err := g.Gather()
	require.NoError(t, err)
	requireGaugeValue(t, metrics, "sentry_organization_projects_total", 2)
}

func TestNumberOfUnresolvedIssues(t *testing.T) {
	logger := zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr}).Level(zerolog.DebugLevel)
	ctx := logger.WithContext(context.Background())
	handler := http.NewServeMux()
	handler.HandleFunc("/api/0/organizations/exists/projects/", func(w http.ResponseWriter, r *http.Request) {
		sendJSONFile(t, w, "projects.json")
	})
	handler.HandleFunc("/api/0/projects/exists/project-1/issues/", func(w http.ResponseWriter, r *http.Request) {
		sendJSONFile(t, w, "issues-project-1.json")
	})
	handler.HandleFunc("/api/0/projects/exists/project-2/issues/", func(w http.ResponseWriter, r *http.Request) {
		sendJSONFile(t, w, "issues-project-2.json")
	})
	srv := httptest.NewServer(handler)
	defer srv.Close()

	g, err := New(Options{
		Token:        "...",
		Organization: "exists",
		APIBaseURL:   srv.URL,
	})
	require.NoError(t, err)
	require.NotNil(t, g)
	g.updateAll(ctx)
	metrics, err := g.Gather()
	require.NoError(t, err)
	requireGaugeValue(t, metrics, "sentry_organization_projects_total", 2)
	requireGaugeValue(t, metrics, "sentry_organization_project_issues_total", 1, labelPair("project", "project-1"))
	requireGaugeValue(t, metrics, "sentry_organization_project_issues_total", 2, labelPair("project", "project-2"))
}

func labelPair(name string, value string) dto.LabelPair {
	return dto.LabelPair{
		Name:  &name,
		Value: &value,
	}
}

func matchesLabels(t *testing.T, metric *dto.Metric, labels []dto.LabelPair) bool {
	if len(labels) != len(metric.GetLabel()) {
		return false
	}
	pendingMatches := len(labels)
	for _, needle := range labels {
		for _, stalk := range metric.GetLabel() {
			if needle.GetName() == stalk.GetName() && needle.GetValue() == stalk.GetValue() {
				pendingMatches = pendingMatches - 1
			}
		}
	}
	return pendingMatches == 0
}

func requireGaugeValue(t *testing.T, mfamilies []*dto.MetricFamily, name string, value float64, labels ...dto.LabelPair) {
	t.Helper()
	for _, family := range mfamilies {
		if family.GetName() != name {
			continue
		}
		for _, metric := range family.Metric {
			if !matchesLabels(t, metric, labels) {
				continue
			}
			require.Equal(t, value, metric.GetGauge().GetValue(), "%s has an unexpected value", family.GetName())
			return
		}
		break
	}
	t.Errorf("%s not found", name)
	t.Fail()
}

func TestNumberOfTeams(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/api/0/organizations/exists/teams/", func(w http.ResponseWriter, r *http.Request) {
		sendJSONFile(t, w, "teams.json")
	})
	srv := httptest.NewServer(handler)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/0/organizations/exists/teams/")
	require.NoError(t, err)
	require.Equal(t, 200, resp.StatusCode)

	g, err := New(Options{
		Token:        "...",
		Organization: "exists",
		APIBaseURL:   srv.URL,
	})
	require.NoError(t, err)
	require.NotNil(t, g)

	count, err := g.gatherOrgTeamsTotal(context.Background())
	require.NoError(t, err)
	require.Equal(t, 1, count)
}

func requireMetric(t *testing.T, metrics []*dto.MetricFamily, name string) *dto.MetricFamily {
	t.Helper()
	for _, m := range metrics {
		if m.GetName() == name {
			return m
		}
	}
	require.FailNow(t, fmt.Sprintf("no metric with the name `%s` found", name))
	return nil
}

func sendJSONFile(t *testing.T, w http.ResponseWriter, name string) {
	t.Helper()
	fp, err := os.Open(filepath.Join("testdata", name))
	require.NoError(t, err)
	defer fp.Close()
	w.Header().Set("Content-Type", "application/json")
	io.Copy(w, fp)
}
