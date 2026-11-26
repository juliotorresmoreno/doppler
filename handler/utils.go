package handler

import (
	"encoding/base64"
	"errors"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/juliotorresmoreno/doppler/config"
	"golang.org/x/crypto/bcrypt"
)

func (h *Proxy) AuthRequired(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Proxy-Authenticate", `Basic realm="Doppler"`)
	w.Header().Add("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusProxyAuthRequired)
	w.Write([]byte(authenticationRequiredHTML))
}

const authenticationRequiredHTML = `<html>
<head><title>Proxy Authentication Required</title></head>
<body>
<h1>Proxy Authentication Required</h1>
<p>This proxy server requires authentication to access the requested resource.</p>
</body>
</html>`

var (
	errUnauthorized = errors.New("Unauthorized")
	errAuthMissing  = errors.New("no credentials")
	errAuthBlocked  = errors.New("blocked after repeated failures")
	errAuthFailed   = errors.New("invalid credentials")
)

// authLimiter blocks an IP after too many failed logins. It is rebuilt when the
// auth limits change on reload; otherwise it keeps its state.
var authLimiter atomic.Pointer[failLimiter]

// limiterFor returns the limiter for the given auth settings.
func limiterFor(a config.Auth) *failLimiter {
	maxFails, window, block := a.MaxFailures, time.Duration(a.FailureWindowMin)*time.Minute, time.Duration(a.BlockMin)*time.Minute
	if maxFails <= 0 {
		maxFails = 5
	}
	if window <= 0 {
		window = 15 * time.Minute
	}
	if block <= 0 {
		block = 15 * time.Minute
	}
	if cur := authLimiter.Load(); cur != nil && cur.maxFails == maxFails && cur.window == window && cur.blockFor == block {
		return cur
	}
	fresh := newFailLimiter(maxFails, window, block)
	authLimiter.Store(fresh)
	return fresh
}

// dummyHash is compared when the user does not exist, so a missing user takes
// the same time as a wrong password.
var (
	dummyOnce sync.Once
	dummyHash []byte
)

// BasicAuth validates a Proxy-Authorization header for the client at clientIP.
// An empty header is not a failure; it only triggers the 407 challenge.
// BasicAuth validates a Proxy-Authorization header for the client at clientIP.
// It returns the authenticated user name. An empty header is not a failure; it
// only triggers the 407 challenge.
func (h *Proxy) BasicAuth(credentials, clientIP string) (string, error) {
	conf, err := config.GetConfig()
	if err != nil {
		return "", errAuthFailed
	}
	if credentials == "" {
		return "", errAuthMissing
	}
	lim := limiterFor(conf.Auth)
	if lim.blocked(clientIP) {
		log.Printf("[AUTH] blocked ip=%s (too many failed attempts)", clientIP)
		return "", errAuthBlocked
	}

	username, password, ok := parseBasic(credentials)
	if !ok {
		authFailed(lim, clientIP, "")
		return "", errAuthFailed
	}

	stored, exists := conf.Auth.Users[username]
	var valid bool
	if exists {
		valid = verifyPassword(stored, password)
	} else {
		dummyOnce.Do(func() {
			dummyHash, _ = bcrypt.GenerateFromPassword([]byte("doppler-dummy"), bcrypt.DefaultCost)
		})
		bcrypt.CompareHashAndPassword(dummyHash, []byte(password))
	}

	if !valid {
		authFailed(lim, clientIP, username)
		return "", errAuthFailed
	}
	lim.reset(clientIP)
	return username, nil
}

// parseBasic extracts user and password from "Basic <base64(user:password)>".
func parseBasic(credentials string) (string, string, bool) {
	const prefix = "basic "
	if len(credentials) <= len(prefix) || !strings.EqualFold(credentials[:len(prefix)], prefix) {
		return "", "", false
	}
	decoded, err := base64.StdEncoding.DecodeString(credentials[len(prefix):])
	if err != nil {
		return "", "", false
	}
	parts := strings.SplitN(string(decoded), ":", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	return parts[0], parts[1], true
}

// verifyPassword accepts only bcrypt hashes. Any other stored format is rejected.
func verifyPassword(stored, password string) bool {
	if !strings.HasPrefix(stored, "$2") {
		return false
	}
	return bcrypt.CompareHashAndPassword([]byte(stored), []byte(password)) == nil
}

func authFailed(lim *failLimiter, clientIP, username string) {
	lim.fail(clientIP)
	log.Printf("[AUTH] failed login user=%q ip=%s", username, clientIP)
}

// clientIP returns the host part of the request's remote address.
func clientIP(req *http.Request) string {
	host, _, err := net.SplitHostPort(req.RemoteAddr)
	if err != nil {
		return req.RemoteAddr
	}
	return host
}
