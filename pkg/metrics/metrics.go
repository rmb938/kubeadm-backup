package metrics

import (
"fmt"
"net"
"net/http"

"github.com/go-logr/logr"
"github.com/prometheus/client_golang/prometheus"
"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	Log logr.Logger
)

// RegistererGatherer combines both parts of the API of a Prometheus
// registry, both the Registerer and the Gatherer interfaces.
type RegistererGatherer interface {
	prometheus.Registerer
	prometheus.Gatherer
}

// Registry is a prometheus registry for storing metrics
var Registry RegistererGatherer = prometheus.NewRegistry()

func ServeMetrics() {
	var metricsPath = "/metrics"
	var addr = ":8080"
	handler := promhttp.HandlerFor(Registry, promhttp.HandlerOpts{
		ErrorHandling: promhttp.HTTPErrorOnError,
	})

	mux := http.NewServeMux()
	mux.Handle(metricsPath, handler)
	mux.Handle("/", http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
		rw.WriteHeader(http.StatusOK)
	}))
	server := http.Server{
		Handler: mux,
	}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		er := fmt.Errorf("error listening on %s: %w", addr, err)
		Log.Error(er, "metrics server failed to listen. You may want to disable the metrics server or use another port if it is due to conflicts")
		return
	}
	// Run the server
	Log.Info("starting metrics server", "path", metricsPath)
	if err := server.Serve(ln); err != nil && err != http.ErrServerClosed {
		Log.Error(err, "server shutdown")
	}
}
