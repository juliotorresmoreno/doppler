package config

import (
	"errors"
	"fmt"
	"strings"
)

// Reload reads the config file again and, if it is valid, makes it the active
// configuration. The new config takes effect for the next request or connection.
//
// Settings the listeners read only at startup cannot change while running: their
// current values are kept, and each attempted change is returned as a warning.
// If the file cannot be read or is invalid, the current configuration stays.
func Reload() ([]string, error) {
	if configPath == "" {
		return nil, errors.New("no configuration loaded")
	}
	next, err := Load(configPath)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", configPath, err)
	}
	if errs := Validate(next); len(errs) > 0 {
		return nil, fmt.Errorf("rejected: %s", strings.Join(errs, "; "))
	}

	cur, err := GetConfig()
	if err != nil {
		return nil, err
	}
	var warnings []string
	keep := func(field string, changed bool) {
		if changed {
			warnings = append(warnings, fmt.Sprintf("%s changed; it needs a restart, keeping the running value", field))
		}
	}
	keep("addr", next.Addr != cur.Addr)
	keep("tls_addr", next.TLSAddr != cur.TLSAddr)
	keep("tls_cert", next.TLSCert != cur.TLSCert)
	keep("tls_key", next.TLSKey != cur.TLSKey)
	keep("metrics_addr", next.MetricsAddr != cur.MetricsAddr)
	keep("max_connections", next.MaxConnections != cur.MaxConnections)
	keep("read_timeout", next.ReadTimeout != cur.ReadTimeout)
	keep("write_timeout", next.WriteTimeout != cur.WriteTimeout)
	keep("max_header_bytes", next.MaxHeaderBytes != cur.MaxHeaderBytes)

	next.Addr = cur.Addr
	next.TLSAddr = cur.TLSAddr
	next.TLSCert = cur.TLSCert
	next.TLSKey = cur.TLSKey
	next.MetricsAddr = cur.MetricsAddr
	next.MaxConnections = cur.MaxConnections
	next.ReadTimeout = cur.ReadTimeout
	next.WriteTimeout = cur.WriteTimeout
	next.MaxHeaderBytes = cur.MaxHeaderBytes

	Set(next)
	return warnings, nil
}
