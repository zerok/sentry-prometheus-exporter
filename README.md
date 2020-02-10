# sentry-prometheus-exporter

This is an experimental exporter for Sentry metrics to be consumed by
Prometheus.

**WARNING:** This is alpha code. *Things are going to change, often, basically
always!*

## Getting started

For now, you have to build the exporter yourself. Configuration happens through
the `SENTRY_TOKEN` environment variable and various commandline arguments:

```
$ make

$ export SENTRY_TOKEN=...

$ ./bin/sentry-prometheus-exporter --organization myorg &

$ curl -i http://localhost:9200
HTTP/1.1 200 OK
Content-Type: text/plain; version=0.0.4; charset=utf-8
Date: Mon, 10 Feb 2020 17:04:22 GMT
Content-Length: 318

# HELP sentry_organization_projects_total Number of projects in an organization
# TYPE sentry_organization_projects_total gauge
sentry_organization_projects_total 4
# HELP sentry_organization_teams_total Number of teams in an organization
# TYPE sentry_organization_teams_total gauge
sentry_organization_teams_total 2

```
