package handler

import (
	"encoding/base64"
	"strings"
	"testing"
)

// FuzzParseBasic: any Proxy-Authorization value must be handled without a panic,
// and an accepted value must decode to exactly user:password.
func FuzzParseBasic(f *testing.F) {
	f.Add("Basic " + base64.StdEncoding.EncodeToString([]byte("julio:secret")))
	f.Add("basic " + base64.StdEncoding.EncodeToString([]byte("a:b:c")))
	f.Add("Basic ")
	f.Add("Basic !!!")
	f.Add("Digest abc")
	f.Add("")
	f.Fuzz(func(t *testing.T, header string) {
		user, pass, ok := parseBasic(header)
		if !ok {
			return
		}
		if len(header) < len("basic ") || !strings.EqualFold(header[:len("basic ")], "basic ") {
			t.Fatalf("accepted a header without the Basic scheme: %q", header)
		}
		raw, err := base64.StdEncoding.DecodeString(header[len("basic "):])
		if err != nil {
			t.Fatalf("accepted undecodable credentials: %q", header)
		}
		if string(raw) != user+":"+pass {
			t.Fatalf("decoded %q, parsed %q and %q", raw, user, pass)
		}
	})
}
