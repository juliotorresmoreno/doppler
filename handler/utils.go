package handler

import (
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/juliotorresmoreno/doppler/config"
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

func (h *Proxy) BasicAuth(credentials string) error {
	conf, err := config.GetConfig()
	if err != nil {
		return errors.New("Unauthorized")
	}

	if len(credentials) < 6 {
		return errors.New("Unauthorized")
	}

	decoded, err := base64.StdEncoding.DecodeString(credentials[6:])
	if err != nil {
		return errors.New("Unauthorized")
	}

	parts := strings.SplitN(string(decoded), ":", 2)
	if len(parts) != 2 {
		return errors.New("Unauthorized")
	}

	username, password := parts[0], parts[1]

	if storedHash, ok := conf.Auth.Users[username]; ok {
		sum := sha256.Sum256([]byte(strings.Trim(password, " ")))
		hashStr := fmt.Sprintf("%x", sum)
		if hashStr == storedHash {
			return nil
		}
	}

	return errors.New("Unauthorized")
}
