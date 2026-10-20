// Command galene-prom-exporter exposes the statistics of a Galene
// videoconferencing server in the Prometheus exposition format.
//
// It acts as a simple proxy: every scrape of /metrics triggers a request
// to Galene's /galene-api/v0/.stats endpoint, and the result is translated
// into Prometheus metrics.
package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func envOr(name, def string) string {
	if v, ok := os.LookupEnv(name); ok {
		return v
	}
	return def
}

func main() {
	var (
		listen       = flag.String("listen", envOr("GALENE_EXPORTER_LISTEN", ":9814"), "`address` to serve metrics on")
		metricsPath  = flag.String("metrics-path", "/metrics", "HTTP `path` to serve metrics on")
		galeneURL    = flag.String("galene-url", envOr("GALENE_URL", "https://localhost:8443"), "base `URL` of the Galene server")
		username     = flag.String("username", os.Getenv("GALENE_USERNAME"), "admin `user` for HTTP Basic authentication (env GALENE_USERNAME)")
		passwordFile = flag.String("password-file", "", "read the admin password from `file` (default: env GALENE_PASSWORD)")
		tokenFile    = flag.String("token-file", "", "read an admin bearer token from `file` (default: env GALENE_TOKEN)")
		insecure     = flag.Bool("insecure", false, "do not verify Galene's TLS certificate")
		timeout      = flag.Duration("timeout", 5*time.Second, "timeout for requests to Galene")
		trackMetrics = flag.Bool("track-metrics", true, "export per-connection and per-track metrics")
	)
	flag.Parse()

	password := os.Getenv("GALENE_PASSWORD")
	if *passwordFile != "" {
		password = mustReadSecret(*passwordFile)
	}
	token := os.Getenv("GALENE_TOKEN")
	if *tokenFile != "" {
		token = mustReadSecret(*tokenFile)
	}

	transport := http.DefaultTransport.(*http.Transport).Clone()
	if *insecure {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}

	collector := &galeneCollector{
		url:      strings.TrimRight(*galeneURL, "/") + "/galene-api/v0/.stats",
		username: *username,
		password: password,
		token:    token,
		client: &http.Client{
			Transport: transport,
			Timeout:   *timeout,
		},
		trackMetrics: *trackMetrics,
	}

	reg := prometheus.NewRegistry()
	reg.MustRegister(
		collector,
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)

	http.Handle(*metricsPath, promhttp.HandlerFor(reg, promhttp.HandlerOpts{
		ErrorLog: log.Default(),
	}))
	if *metricsPath != "/" {
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/" {
				http.NotFound(w, r)
				return
			}
			fmt.Fprintf(w, "<html><head><title>Galene Exporter</title></head>"+
				"<body><h1>Galene Exporter</h1>"+
				"<p><a href=\"%s\">Metrics</a></p></body></html>\n",
				*metricsPath)
		})
	}

	log.Printf("Exporting %v on %v%v", collector.url, *listen, *metricsPath)
	log.Fatal(http.ListenAndServe(*listen, nil))
}

func mustReadSecret(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		log.Fatalf("Read secret: %v", err)
	}
	return strings.TrimSpace(string(b))
}
