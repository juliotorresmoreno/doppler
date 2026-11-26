package handler

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
)

func TestParseBasic(t *testing.T) {
	good := "Basic " + base64.StdEncoding.EncodeToString([]byte("alice:s3cret:x"))
	u, p, ok := parseBasic(good)
	if !ok || u != "alice" || p != "s3cret:x" {
		t.Fatalf("parseBasic(good) = %q %q %v", u, p, ok)
	}
	if _, _, ok := parseBasic("basic " + base64.StdEncoding.EncodeToString([]byte("a:b"))); !ok {
		t.Error("scheme should be case-insensitive")
	}
	for _, bad := range []string{"", "Basic", "Digest abc", "Basic !!!", "Basic " + base64.StdEncoding.EncodeToString([]byte("nocolon"))} {
		if _, _, ok := parseBasic(bad); ok {
			t.Errorf("parseBasic(%q) accepted", bad)
		}
	}
}

func TestVerifyPasswordBcrypt(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("correct horse"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	if !verifyPassword(string(hash), "correct horse") {
		t.Error("correct password rejected")
	}
	if verifyPassword(string(hash), "wrong") {
		t.Error("wrong password accepted")
	}
	if verifyPassword(string(hash), "correct horse ") {
		t.Error("bcrypt must not trim spaces")
	}
}

func TestVerifyPasswordRejectsLegacySHA256(t *testing.T) {
	sum := sha256.Sum256([]byte("admin123"))
	legacy := hex.EncodeToString(sum[:])
	if verifyPassword(legacy, "admin123") {
		t.Error("unsalted SHA-256 hashes must no longer be accepted")
	}
}

func TestVerifyPasswordUnknownFormat(t *testing.T) {
	if verifyPassword("plaintext", "plaintext") {
		t.Error("plaintext stored password must never be accepted")
	}
}

func TestFailLimiterBlocksAfterMaxFailures(t *testing.T) {
	l := newFailLimiter(3, time.Minute, time.Minute)
	for i := 0; i < 2; i++ {
		l.fail("10.0.0.1")
	}
	if l.blocked("10.0.0.1") {
		t.Fatal("blocked too early")
	}
	l.fail("10.0.0.1")
	if !l.blocked("10.0.0.1") {
		t.Fatal("should be blocked after max failures")
	}
	if l.blocked("10.0.0.2") {
		t.Fatal("other IPs must not be affected")
	}
	l.reset("10.0.0.1")
	if l.blocked("10.0.0.1") {
		t.Fatal("reset should unblock")
	}
}
