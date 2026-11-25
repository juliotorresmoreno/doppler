package server

import (
	"bytes"
	"net"
	"net/http"
	"os"
	"text/template"

	"github.com/juliotorresmoreno/doppler/config"
)

// servePAC renders proxy.pac for the host the client used to reach Doppler,
// unless public_host is set. The file is read on every request so edits apply
// without a restart. The PAC always points at the TLS listener.
func servePAC(w http.ResponseWriter, r *http.Request) {
	// The PAC decides where all browser traffic goes, so over plain HTTP it is
	// served only to clients on this machine (loopback). Remote clients need TLS.
	if r.TLS == nil && !isLoopback(r.RemoteAddr) {
		http.NotFound(w, r)
		return
	}
	conf, err := config.GetConfig()
	if err != nil || conf.TLSAddr == "" {
		http.Error(w, "TLS listener is not enabled", http.StatusServiceUnavailable)
		return
	}
	_, tlsPort, err := net.SplitHostPort(conf.TLSAddr)
	if err != nil {
		http.Error(w, "invalid tls_addr", http.StatusInternalServerError)
		return
	}

	host := conf.PublicHost
	if host == "" {
		host = r.Host
		if h, _, err := net.SplitHostPort(r.Host); err == nil {
			host = h
		}
	}

	tmplBody, err := os.ReadFile(config.PACFile())
	if err != nil {
		http.Error(w, "proxy.pac not found", http.StatusNotFound)
		return
	}
	tmpl, err := template.New("pac").Parse(string(tmplBody))
	if err != nil {
		http.Error(w, "invalid proxy.pac", http.StatusInternalServerError)
		return
	}
	var out bytes.Buffer
	data := struct{ Host, TLSPort string }{Host: host, TLSPort: tlsPort}
	if err := tmpl.Execute(&out, data); err != nil {
		http.Error(w, "invalid proxy.pac", http.StatusInternalServerError)
		return
	}

	// text/plain + inline so browsers display the file instead of downloading it.
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Content-Disposition", "inline; filename=\"proxy.pac\"")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", "no-store")
	w.Write(out.Bytes())
}

func isLoopback(remoteAddr string) bool {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		return false
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
