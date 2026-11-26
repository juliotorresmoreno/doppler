package handler

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"

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

var errUnauthorized = errors.New("Unauthorized")

// authLimiter blocks an IP after too many failed logins.
var authLimiter = newFailLimiter(maxAuthFailures, authFailWindow, authBlockDuration)

// dummyHash is compared when the user does not exist, so a missing user takes
// the same time as a wrong password.
var (
	dummyOnce sync.Once
	dummyHash []byte
)

// BasicAuth validates a Proxy-Authorization header for the client at clientIP.
// An empty header is not a failure; it only triggers the 407 challenge.
func (h *Proxy) BasicAuth(credentials, clientIP string) error {
	conf, err := config.GetConfig()
	if err != nil {
		return errUnauthorized
	}
	if credentials == "" {
		return errUnauthorized
	}
	if authLimiter.blocked(clientIP) {
		log.Printf("[AUTH] blocked ip=%s (too many failed attempts)", clientIP)
		return errUnauthorized
	}

	username, password, ok := parseBasic(credentials)
	if !ok {
		authFailed(clientIP, "")
		return errUnauthorized
	}

	stored, exists := conf.Auth.Users[username]
	var valid bool
	if exists {
		valid = verifyPassword(username, stored, password)
	} else {
		dummyOnce.Do(func() {
			dummyHash, _ = bcrypt.GenerateFromPassword([]byte("doppler-dummy"), bcrypt.DefaultCost)
		})
		bcrypt.CompareHashAndPassword(dummyHash, []byte(password))
	}

	if !valid {
		authFailed(clientIP, username)
		return errUnauthorized
	}
	authLimiter.reset(clientIP)
	return nil
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

// verifyPassword accepts bcrypt hashes. Legacy unsalted SHA-256 hashes are
// still accepted so existing configs keep working, with a warning to re-hash.
func verifyPassword(username, stored, password string) bool {
	if strings.HasPrefix(stored, "$2") {
		return bcrypt.CompareHashAndPassword([]byte(stored), []byte(password)) == nil
	}
	if len(stored) == sha256.Size*2 {
		if _, err := hex.DecodeString(stored); err == nil {
			// Legacy hashes were computed on the password with surrounding spaces trimmed.
			sum := sha256.Sum256([]byte(strings.Trim(password, " ")))
			ok := subtle.ConstantTimeCompare([]byte(hex.EncodeToString(sum[:])), []byte(strings.ToLower(stored))) == 1
			if ok {
				log.Printf("[AUTH] user %q uses a legacy SHA-256 hash; regenerate it with cmd/hashpw", username)
			}
			return ok
		}
	}
	return false
}

func authFailed(clientIP, username string) {
	authLimiter.fail(clientIP)
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
