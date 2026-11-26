package utils

import (
	"fmt"
	"io"
	"net"

	"github.com/juliotorresmoreno/doppler/config"
	"net/http"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
)

// ActiveConnections is the number of proxied connections in progress.
var ActiveConnections atomic.Int64

var (
	metricsMu sync.Mutex
	// counters: metric name -> label set -> value
	counters = map[string]map[string]int64{}
)

// Reasons that count as authentication failures and as connection errors.
var (
	authFailureReasons = map[string]bool{"auth_missing": true, "auth_blocked": true, "auth_failed": true}
	errorReasons       = map[string]bool{"upstream_error": true, "internal_error": true}
)

func incCounter(name, labels string, n int64) {
	metricsMu.Lock()
	defer metricsMu.Unlock()
	if counters[name] == nil {
		counters[name] = map[string]int64{}
	}
	counters[name][labels] += n
}

// ObserveAccess updates the counters from one finished request.
//
// Counted: requests by kind, allow/deny decisions by reason (DNS failures are
// the reason "dns_failure"), authentication failures, connection errors, and
// bytes transferred in each direction.
func ObserveAccess(ev *AccessEvent) {
	incCounter("doppler_requests_total", fmt.Sprintf(`event="%s"`, ev.Event), 1)
	if ev.Decision != "" {
		incCounter("doppler_decisions_total", fmt.Sprintf(`decision="%s",reason="%s"`, ev.Decision, ev.Reason), 1)
	}
	if authFailureReasons[ev.Reason] {
		incCounter("doppler_auth_failures_total", fmt.Sprintf(`reason="%s"`, ev.Reason), 1)
	}
	if errorReasons[ev.Reason] {
		incCounter("doppler_connection_errors_total", fmt.Sprintf(`reason="%s"`, ev.Reason), 1)
	}
	if ev.BytesUp > 0 {
		incCounter("doppler_bytes_total", `direction="up"`, ev.BytesUp)
	}
	if ev.BytesDown > 0 {
		incCounter("doppler_bytes_total", `direction="down"`, ev.BytesDown)
	}
}

// WriteMetrics writes all metrics in the Prometheus text format.
func WriteMetrics(w io.Writer) {
	fmt.Fprintf(w, "# TYPE doppler_connections_active gauge\ndoppler_connections_active %d\n", ActiveConnections.Load())

	metricsMu.Lock()
	defer metricsMu.Unlock()
	names := make([]string, 0, len(counters))
	for name := range counters {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		fmt.Fprintf(w, "# TYPE %s counter\n", name)
		labels := make([]string, 0, len(counters[name]))
		for l := range counters[name] {
			labels = append(labels, l)
		}
		sort.Strings(labels)
		for _, l := range labels {
			if l == "" {
				fmt.Fprintf(w, "%s %d\n", name, counters[name][l])
			} else {
				fmt.Fprintf(w, "%s{%s} %d\n", name, l, counters[name][l])
			}
		}
	}
}

// MetricsHandler serves the metrics. When metrics_allow is not empty (read on
// each request, so reloads apply), only clients whose address matches one of its
// IPs or CIDR ranges get the metrics; others get 404.
func MetricsHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conf, _ := config.GetConfig()
		if allow := conf.MetricsAllow; len(allow) > 0 {
			host, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				host = r.RemoteAddr
			}
			ip := net.ParseIP(host)
			if ip == nil || !matchIP(ip, allow) {
				http.NotFound(w, r)
				return
			}
		}
		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		var b strings.Builder
		WriteMetrics(&b)
		io.WriteString(w, b.String())
	})
}
