package utils

import (
	"encoding/json"
	"io"
	"os"
	"sync"
	"time"
)

// AccessEvent is one structured access-log line: one per proxied request. It
// never carries credentials, request paths, query strings or payloads.
type AccessEvent struct {
	Time       string   `json:"time"`
	Event      string   `json:"event"` // connect | http | websocket
	Client     string   `json:"client"`
	User       string   `json:"user,omitempty"`
	Method     string   `json:"method"`
	Dest       string   `json:"dest"`
	Resolved   []string `json:"resolved,omitempty"`
	Decision   string   `json:"decision"` // allow | deny
	Reason     string   `json:"reason"`
	Status     int      `json:"status"`
	DurationMS int64    `json:"duration_ms"`
	BytesUp    int64    `json:"bytes_up"`
	BytesDown  int64    `json:"bytes_down"`

	start time.Time
}

// AccessOutput receives the JSON lines. Stdout, so journald collects them.
var AccessOutput io.Writer = os.Stdout

var accessMu sync.Mutex

// NewAccessEvent starts an event for a request. Call Emit when it ends.
func NewAccessEvent(event, client, method, dest string) *AccessEvent {
	now := time.Now()
	return &AccessEvent{
		Time:   now.Format(time.RFC3339),
		Event:  event,
		Client: client,
		Method: method,
		Dest:   dest,
		start:  now,
	}
}

// SetDecision records the ACL outcome and the addresses it resolved.
func (e *AccessEvent) SetDecision(allowed bool, reason string, ips []string) {
	if allowed {
		e.Decision = "allow"
	} else {
		e.Decision = "deny"
	}
	e.Reason = reason
	e.Resolved = ips
}

// Emit writes the event as one JSON line.
func (e *AccessEvent) Emit() {
	e.DurationMS = time.Since(e.start).Milliseconds()
	line, err := json.Marshal(e)
	if err != nil {
		return
	}
	accessMu.Lock()
	defer accessMu.Unlock()
	AccessOutput.Write(append(line, '\n'))
}
