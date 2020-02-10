all: bin/sentry-prometheus-exporter

bin:
	mkdir -p bin

bin/sentry-prometheus-exporter: bin go.mod $(shell find . -name '*.go')
	cd cmd/sentry-prometheus-exporter && go build -o ../../$@

test:
	go test ./... -v

clean:
	rm -rf bin

.PHONY: clean all test
