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

	count, err := g.gatherOrgProjectsTotal(context.Background())
	require.NoError(t, err)
	require.Equal(t, 2, count)
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
