# Doppler — Technical Review & Improvement Report

**Project:** Doppler HTTP CONNECT Proxy  
**Language:** Go  
**Review status:** Implemented; status tracked per item (updated 2026-11-26)  
**Objective:** Identify security, correctness, reliability, and product improvements before expanding Doppler's capabilities.

---

## 1. Critical: Security

### 1.1 TLS for client-to-proxy communication

**Status: ✅ Done.** `tls_addr` serves the proxy over TLS with `tls_cert`/`tls_key`, validated at startup. With auth enabled, credentials are refused on the plain listener, and the service does not start without TLS. *Evidence:* `main.go`, `handler/proxy.go`, `TestAuthOverTLS`, `TestAuthRejectedOnPlainHTTP`.
*Observation:* the installer's certificate is self-signed for `localhost`/`127.0.0.1`. Mutual TLS is not implemented (the report marks it optional).

**Current situation**

The client-to-proxy connection does not currently use TLS.

Proxy authentication and requests may therefore be transmitted without transport encryption.

**Proposed improvements**

- Support HTTPS proxy connections.
- Add TLS certificate and private key configuration.
- Validate TLS configuration.
- Consider optional mutual TLS for deployments requiring stronger client authentication.

**Priority:** Critical.

### 1.2 Authentication

**Status: ✅ Done, with one item partial.** Passwords are bcrypt-hashed with per-hash salt; the SHA-256 path was removed; failed logins are rate-limited per IP; the limits and the password policy are configurable; failures are logged with user and IP. *Evidence:* `handler/utils.go`, `helper/password.go`, `cmd/hashpw`, `handler/auth_test.go`, `TestAccessLogFailedAuthRecordsReason`.
*Partial:* auth is still **off by default**. Startup and config validation catch a broken auth config, but nothing forces auth on in production.

**Current situation**

Doppler supports authentication, with passwords stored as SHA-256 hashes.

**Things to review**

- Whether passwords are hashed with a unique salt.
- Whether a password-specific hashing algorithm should replace plain SHA-256.
- Whether credentials are protected against brute-force attempts.
- Whether authentication can be disabled accidentally in production.
- Whether authentication failures are logged appropriately.

**Proposed improvements**

- Use a password hashing algorithm such as Argon2id or bcrypt for stored credentials.
- Add configurable authentication requirements.
- Add protection against repeated failed authentication attempts.

**Priority:** High.

---

## 2. Critical: Access Control and DNS

### 2.1 DNS resolution failures

**Status: ✅ Done.** An unresolvable name is denied unless it is explicitly permitted, whatever the default policy. *Evidence:* `utils.Decide` (`ReasonDNSFailure`), `TestDNSFailureDeniesUnlessPermitted`, live check.

**Current situation**

The access-control function resolves the requested hostname and evaluates the resulting IP addresses.

A potential problem exists if DNS resolution fails while the default policy is `permit`.

**Proposed improvement**

Define explicit behavior for DNS resolution failures.

For restrictive environments, DNS resolution failures should result in denied connections.

**Priority:** Critical.

### 2.2 Multiple resolved IP addresses

**Status: ✅ Done.** A block on the name or on any resolved address denies. The precedence is documented in the README ("Policy evaluation"). *Evidence:* `TestDomainWithMultipleAddressesBlockedByOneAddress`, `TestBlockedIPDeniesEvenIfNamePermitted`.

**Current situation**

The current implementation evaluates hostnames and resolved IP addresses individually.

Because it returns immediately when it finds a permitted match, a hostname with multiple resolved IPs may be allowed even if another resolved IP is blocked.

**Proposed improvement**

Define an unambiguous evaluation policy.

For example:

- Explicit block matches always deny the connection.
- Explicit permit matches are evaluated according to a documented precedence policy.
- Unmatched destinations follow the default policy.

**Priority:** Critical.

### 2.3 Domain matching

**Status: ✅ Done.** Exact names, `*.domain` wildcards (subdomains only, not the apex), case-insensitive, trailing dot ignored, internationalized names compared in punycode. The rules are documented in the README. *Evidence:* `internal/utils/match.go`, `TestWildcardThroughProxy`, `TestInternationalizedNameThroughProxy`.
*Observation:* IDN is tested through the proxy with punycode requests only. Go's server rejects non-ASCII CONNECT targets, so Unicode in the request line is not a case to support.

**Current situation**

The current `strings.EqualFold` implementation performs case-insensitive exact matching.

It does not implement wildcard matching or automatic subdomain matching.

**Proposed improvements**

- Document exact hostname matching.
- Decide whether wildcard rules are necessary.
- Consider explicit subdomain matching.
- Normalize hostnames before evaluating policies.
- Add tests for case differences, trailing dots, and internationalized domain names.

**Priority:** High.

### 2.4 IP-based access control

**Status: ✅ Done.** IPv4 and IPv6, CIDR ranges in `permit` and `block`, IPv4-mapped IPv6 handled as IPv4, special-purpose ranges (loopback, private, link-local, multicast, reserved) denied by default (`block_special_ranges`). Rebinding and time-of-check/time-of-use are addressed by resolving once and dialing only approved addresses (see 2.5 and 3.3). *Evidence:* `isSpecial`, `TestSpecialRangeDeniedOverTunnel`, `TestIPv4MappedIPv6LiteralIsMatchedAsIPv4`.
*Observation:* the system resolver is trusted; a resolver that lies can still steer a name to an allowed address. DNS-over-TLS is out of scope.

**Current situation**

Doppler resolves hostnames and checks their associated IP addresses.

**Things to review**

- IPv4 and IPv6 support.
- CIDR range support.
- IPv4-mapped IPv6 addresses.
- Private, loopback, and link-local address handling.
- DNS rebinding and time-of-check/time-of-use scenarios.

**Proposed improvements**

Consider adding CIDR-based policies and explicit handling of special-purpose IP ranges.

**Priority:** High.

### 2.5 Direct IP connections

**Status: ✅ Done.** Clients can CONNECT by literal IP. Those requests go through the same policy and special-range check. Alternate notations (`2130706433`, `127.1`) are not parsed as IPs, so they are treated as unresolvable names and denied. *Evidence:* `TestDirectIPConnect`, live `403` on `127.0.0.1`.

**Things to review**

Determine whether clients can issue CONNECT requests using literal IP addresses instead of hostnames.

If so, ensure that these requests are evaluated against the same access-control policies.

**Priority:** High.

---

## 3. Proxy Reliability

### 3.1 Connection lifecycle

**Status: ✅ Done.** Both sides are closed when a tunnel ends; half-close is propagated; idle and maximum tunnel durations are enforced; goroutines return to baseline; shutdown closes open tunnels (SIGTERM/SIGINT) before stopping the listeners; a failed hijack closes the destination socket. *Evidence:* `internal/utils/relay.go`, `handler/tunnels.go`, `handler/lifecycle_test.go`, `main_test.go`.
*Note:* cancellation is done through a tunnel registry rather than `context.Context`, which is equivalent for this design.

Review how Doppler handles:

- Client disconnections.
- Destination disconnections.
- Half-closed TCP connections.
- Context cancellation.
- Goroutine cleanup.
- Socket closure.
- Connection timeouts.

**Priority:** High.

### 3.2 Resource limits

**Status: ✅ Done.** Configurable: `max_connections` (503 when full), `max_tunnel_seconds`, `idle_timeout` (also the HTTP keep-alive timeout), `max_header_bytes`, `dial_timeout`, bandwidth `limit` per direction, auth attempt limits. The service unit sets `LimitNOFILE=8192`. *Evidence:* `TestResourceLimitMaxConnections`, `TestResourceLimitMaxTunnelDuration`, `TestMaxHeaderBytesIsEnforced`.
*Observation:* memory is bounded by the connection cap (two 32 KB buffers per direction per tunnel), but it has not been measured under load.

Review existing transfer limits and determine whether additional controls are necessary:

- Maximum concurrent connections.
- Maximum connection duration.
- Idle connection timeout.
- Maximum request header size.
- Maximum authentication attempts.
- Memory and file descriptor usage.

**Priority:** Medium.

### 3.3 Connection reuse

**Status: ✅ Done (decision: no reuse).** Each CONNECT gets its own destination connection. Forwarded HTTP requests used to leave an idle transport open for 90 seconds; they now close their connections when the request ends (`TestForwardedRequestsCloseTheirConnections`). Pooling was considered and rejected: it adds complexity and would reuse an authorization made for another request or client. *Evidence:* README, "Tunnel lifecycle".

Investigate whether Doppler creates independent TCP connections for each CONNECT request and whether any connection reuse exists.

HTTP CONNECT normally establishes a dedicated tunnel, so connection pooling should not be assumed to be beneficial or appropriate.

**Priority:** Low / Investigation.

---

## 4. Observability and Administration

### 4.1 Access logs

**Status: ✅ Done.** One JSON line per request on stdout (`access_log`, on by default), with time, event, client, user, destination, resolved IPs, decision, reason, status, duration and bytes. Passwords, `Proxy-Authorization` and paths are never logged. *Evidence:* `internal/utils/access.go`, `handler/access_test.go`, live journal line.

Add structured logs for:

- Connection attempts.
- Authenticated user.
- Requested destination.
- Resolved IP addresses.
- Allow or deny decision.
- Rejection reason.
- Connection duration.
- Transferred bytes.

Avoid logging passwords, authorization headers, or sensitive payloads.

**Priority:** High.

### 4.2 Metrics

**Status: ✅ Done.** Prometheus text on a separate listener (`metrics_addr`), optionally restricted by `metrics_allow`: active connections (gauge), requests by kind, allow/deny decisions by reason (DNS failures appear as `dns_failure`), authentication failures, connection errors, bytes per direction. *Evidence:* `internal/utils/metrics.go`, `server/server_test.go`, `handler/metrics_test.go`.
*Observation:* counters are in memory and reset on restart. Requests are counted when they finish, so a long tunnel shows in `connections_active` before it appears in the totals.

Consider exposing:

- Active connections.
- Total connections.
- Allowed connections.
- Blocked connections.
- Authentication failures.
- DNS resolution failures.
- Transferred bytes.
- Connection errors.

**Priority:** Medium.

### 4.3 Configuration management

**Status: ✅ Done.** Startup validation lists every problem and refuses to start. `SIGHUP` reloads the file: an invalid file is rejected and the running config stays; the policy, auth and limits swap atomically; settings that need a restart are kept and reported. *Evidence:* `config/validate.go`, `config/reload.go`, `main_test.go` (`TestReloadAndGracefulShutdown`).

Investigate whether configuration changes require restarting the proxy.

Potential improvements:

- Configuration validation.
- Better startup error messages.
- Safe configuration reload.
- Atomic policy updates.

**Priority:** Medium.

---

## 5. Testing

**Status: ✅ Done for the required list.** All 16 required cases have automated tests, and the suite (89 tests plus fuzz seeds) passes with `-race`. CI is defined in `.github/workflows/ci.yml`. *Evidence:* README, "Testing".
*Observations:* the IPv6 destination test skips on machines without IPv6 loopback. The destination-timeout test measures a real 1 s timeout on this machine, but on a network that rejects the test address immediately it would only see the `503`. CI has not been run on GitHub yet, and fuzzing runs only its seed corpus in CI.

Doppler needs a comprehensive automated test suite, especially around access-control behavior.

### Required test cases

- Explicitly permitted domain.
- Explicitly blocked domain.
- Unknown domain under default-deny policy.
- Unknown domain under default-permit policy.
- Domain present in both lists.
- Domain resolving to multiple IP addresses.
- DNS resolution failure.
- Direct IP connection.
- IPv4 and IPv6 destinations.
- Case-insensitive domain matching.
- Subdomain matching behavior.
- Authentication success and failure.
- Invalid CONNECT requests.
- Client disconnection during an active tunnel.
- Destination timeout.
- Concurrent connections.

**Priority:** Critical.

---

## 6. Product Direction

**Status:** Phase 1 done. Phase 2 partly done (validation, reload, metrics). Phase 3 not started.

Doppler currently works as a CLI-oriented proxy utility.

Before building a graphical interface, establish whether the project should remain a lightweight infrastructure tool or evolve into a broader network access-control product.

### Recommended direction

**Phase 1 — Engineering foundation**

- Fix security issues.
- Correct DNS and access-control behavior.
- Improve automated tests.
- Document configuration.
- Improve logs and operational behavior.

**Phase 2 — Operational usability**

- Better CLI commands.
- Configuration validation.
- Policy inspection.
- Runtime configuration reload.
- Metrics.

**Phase 3 — Product exploration**

- Administrative API.
- Web management interface.
- User and policy management.
- Access history and reporting.

A web interface should be optional rather than a requirement for using Doppler.

---

## 7. Recommended Implementation Order

| Priority | Task | Category | Status |
|---|---|---|---|
| P0 | TLS support | Security | ✅ Done |
| P0 | DNS failure behavior | Access control | ✅ Done |
| P0 | Multiple IP evaluation | Access control | ✅ Done |
| P0 | Automated policy tests | Reliability | ✅ Done (CI defined, not yet run on GitHub) |
| P1 | Authentication review | Security | ✅ Done (auth still off by default) |
| P1 | Direct IP handling | Access control | ✅ Done |
| P1 | Structured access logs | Observability | ✅ Done |
| P1 | Connection lifecycle review | Reliability | ✅ Done |
| P2 | CIDR policies | Access control | ✅ Done |
| P2 | Configuration reload | Operations | ✅ Done (SIGHUP) |
| P2 | Metrics | Observability | ✅ Done (Prometheus text) |
| P3 | Administrative API | Product | ❌ Open |
| P3 | Web dashboard | Product | ❌ Open |

---|---|---|
| P0 | TLS support | Security |
| P0 | DNS failure behavior | Access control |
| P0 | Multiple IP evaluation | Access control |
| P0 | Automated policy tests | Reliability |
| P1 | Authentication review | Security |
| P1 | Direct IP handling | Access control |
| P1 | Structured access logs | Observability |
| P1 | Connection lifecycle review | Reliability |
| P2 | CIDR policies | Access control |
| P2 | Configuration reload | Operations |
| P2 | Metrics | Observability |
| P3 | Administrative API | Product |
| P3 | Web dashboard | Product |

---

## 8. Review Principles

The objective is not to turn Doppler into a complex enterprise platform unnecessarily.

The goal is to preserve its original value:

- Lightweight.
- Easy to deploy.
- Configurable.
- Predictable.
- Secure.
- Useful for controlling outbound network access.

Every new feature should justify its operational and architectural complexity.

**Next review:** Validate each finding against the source code, reproduce the relevant behaviors with tests, and prioritize implementation based on actual risks.

---

## 9. Observations and Open Items

Items that remain open or that need a decision, collected while implementing the report:

- **Auth off by default.** Enabling it is a deployment decision; nothing enforces it.
- **Installer overwrites the live config.** `install.sh` copies the repository's `config.yml` to `/etc/doppler/` on every run, so local edits to the live config are lost on reinstall. The existing `cp` lines were not changed.
- **CI not verified remotely.** `.github/workflows/ci.yml` is defined but has not been run on GitHub.
- **Fuzzing in CI is seed-only.** Long fuzzing runs are manual (commands in the README).
- **No CLI for policy inspection.** Phase 2 includes "policy inspection" and "better CLI commands"; neither exists.
- **Metrics endpoint has no authentication.** Protection is the allow list and the network; add TLS and credentials if the port must be reachable from untrusted networks.
- **Memory under load not measured.** The limits bound it, but there has been no load test.
- **Live checks are manual.** Some behavior (the installed service, SIGHUP on `/etc`, the metrics port from the network, browser and VS Code certificate trust) is verified by hand, not by tests.
- **Behavior changes to communicate.** Special-purpose addresses are denied by default; forwarded HTTP connections close after each request; bandwidth `limit` now applies in both directions; a CONNECT without a port returns `400`.
