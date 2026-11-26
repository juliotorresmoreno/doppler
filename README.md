# Doppler

**A lightweight HTTP CONNECT proxy for controlled outbound network access.**

Doppler is an HTTP CONNECT proxy written in Go that allows administrators to control outbound connections through configurable domain and IP-based access policies.

It was originally conceived for environments where outbound network traffic needs to be restricted by default, allowing connections only to explicitly authorized destinations.

Doppler can also be used as a personal proxy to route internet traffic through a remote server, such as an AWS instance.

## Why Doppler?

In environments with strict network requirements, controlling outbound traffic is just as important as protecting incoming connections.

Applications may need access to specific external services while all other destinations remain restricted.

Doppler addresses this problem by providing a centralized proxy with configurable access rules.

Instead of relying exclusively on application-level restrictions, outbound connections can be evaluated against a centralized policy.

## Key Features

- **HTTP CONNECT proxy:** tunnels to remote destinations, plus plain HTTP and WebSocket forwarding.
- **Domain policies:** exact names, `*.example.com` wildcards, case-insensitive, trailing dot ignored, internationalized names.
- **IP policies:** single addresses and CIDR ranges, IPv4 and IPv6, with special-purpose ranges (loopback, private, link-local) blocked by default.
- **Explicit precedence:** block always wins; unresolvable names are denied; the default policy applies to everything else.
- **DNS-safe connections:** each name is resolved once, and the connection goes only to addresses that passed the policy.
- **Authentication:** bcrypt-hashed users, only accepted over TLS, with a per-IP lockout after repeated failures.
- **Limits:** concurrent connections, tunnel duration, idle timeout, header size, dial timeout, bandwidth per direction.
- **Access logs:** one structured JSON line per request, with the user, destination, resolved IPs, decision, reason, duration and bytes. Credentials are never logged.
- **Metrics:** Prometheus format on a separate listener, optionally restricted by client address.
- **Proxy auto-configuration:** serves a PAC file so browsers pick the proxy automatically.
- **Operations:** YAML configuration validated at startup, reloaded with `SIGHUP`, graceful shutdown that closes open tunnels, and an installer with systemd service.

## How It Works

Doppler acts as an intermediary between client applications and external destinations.

When a client requests a connection through the proxy using HTTP CONNECT, Doppler evaluates the requested destination against its access policies.

The proxy resolves the hostname, evaluates the hostname and its resolved IP addresses against the configured rules, and decides whether the connection should be established.

```text
Client Application
        |
        v
   Doppler Proxy
        |
        v
  Authentication
        |
        v
  Access Policies
        |
        +---- Block ----> Connection Rejected
        |
        +---- Permit ---> Destination Connection
                              |
                              v
                           Internet
```

## Access Control

Doppler supports three policy components:

- `permit`: destinations explicitly allowed.
- `block`: destinations explicitly denied.
- `defaultPolicy` (`ACL.default`): the decision applied when no rule matches. Use `block` for restrictive environments.

Configuration lives under `ACL` in `config.yml`:

```yaml
ACL:
  default: block
  permit:
    - example.com          # exact name only
    - "*.example.com"      # any subdomain of example.com
    - 203.0.113.10         # single IP address
    - 10.0.0.0/8          # CIDR range
  block:
    - "*.blocked.example"
    - 198.51.100.0/24
```

### Entry formats

| Entry | Matches |
|---|---|
| `example.com` | exactly `example.com`; **not** its subdomains |
| `*.example.com` | any subdomain at any depth (`a.example.com`, `a.b.example.com`); **not** `example.com` itself |
| `203.0.113.10` | that IP address |
| `10.0.0.0/8` | every address in the range (IPv4 or IPv6) |

Names are case-insensitive, a trailing dot is ignored, and internationalized names match their punycode form (`münchen.de` and `xn--mnchen-3ya.de` are the same name). A wildcard is only accepted as a leading `*.`; `*` alone matches nothing. A subdomain is never matched by an exact name, so list `example.com` and `*.example.com` both if you want the apex and its subdomains.

Name entries are compared with the requested hostname. Address entries (single IP or CIDR) are compared with **every address the hostname resolves to**, and with the address itself when the request is for an IP literal.

### Special-purpose addresses

With `block_special_ranges: true` (the default), addresses that are not meant to be reached through a public proxy are denied, even under `default: permit`:

- loopback (`127.0.0.0/8`, `::1`)
- private (`10/8`, `172.16/12`, `192.168/16`, `fc00::/7`)
- link-local (`169.254/16`, `fe80::/10`), including cloud metadata endpoints such as `169.254.169.254`
- multicast, unspecified (`0.0.0.0`, `::`)
- reserved and documentation ranges (`100.64/10`, `240/4`, `192.0.2/24`, `198.51.100/24`, `203.0.113/24`, and similar)

A special address is allowed only when it is explicitly permitted by IP or CIDR (for example `permit: [127.0.0.1]`), or when the name it belongs to is explicitly permitted. Set `block_special_ranges: false` to turn this off.

### Policy evaluation

Rules are evaluated in this order. The first matching step decides:

1. **Block on name:** the requested hostname matches a `block` entry → **deny**.
2. **Resolve once:** the hostname is resolved one time. The resolved addresses are used for every later check and for the connection.
3. **Block on address:** any resolved address matches a `block` entry (single IP or CIDR) → **deny**. This applies even when the name is permitted, and even when only one of several addresses is blocked.
4. **Permit on name:** the hostname matches a `permit` entry → **allow**.
5. **DNS failure:** the hostname cannot be resolved and was not allowed in step 4 → **deny**, regardless of `defaultPolicy`. Explicitly permitted names still go through, and the connection then fails with an error.
6. **Special-purpose address:** with `block_special_ranges` on, any resolved address in a special range that is not explicitly permitted by IP → **deny**.
7. **Permit on address:** any resolved address matches a `permit` entry → **allow**.
8. **Default:** otherwise `defaultPolicy` applies.

Block always beats permit. Connections are made only to the addresses that passed step 3, so DNS cannot answer differently between the check and the connection.

These rules apply to CONNECT tunnels, plain HTTP forwarding, and WebSocket requests.

**Limitations:** the decision is based on the DNS answer at request time. Hosts behind shared IPs (CDNs, load balancers) are matched by those shared addresses, so an address rule can block or allow more than one site. Test the policy against the destinations you care about before relying on it.

## Tunnel lifecycle

For CONNECT tunnels and WebSocket connections, Doppler:

- **Closes both sockets** when the tunnel ends, so a closed client also closes the destination connection, and the other way around.
- **Half-closes** when one side finishes sending: the peer sees EOF but can still send its reply.
- **Closes idle tunnels** after `idle_timeout` seconds with no traffic (default 300; `0` disables this).
- **Applies `limit` (KB/s) to each direction.** Traffic in both directions is throttled.
- **Connects only to the addresses the ACL approved** (see "Policy evaluation").
- **Does not reuse destination connections:** each CONNECT tunnel, and each forwarded HTTP request, opens its own connection to the destination, and closes it when the request ends. Nothing is pooled, so a connection approved for one request is never reused for another.
- **Shuts down cleanly:** on SIGINT or SIGTERM, Doppler refuses new tunnels, closes the open ones (both sides see the close), and then stops the HTTP listeners, waiting up to 5 seconds for in-flight requests.

Destination connection attempts time out after 10 seconds.

## Access logs

With `access_log: true` (the default), Doppler writes one JSON line per proxied request to stdout. Under systemd these lines go to the journal: `journalctl -u doppler -f`.

```json
{"time":"2026-11-25T23:58:01-05:00","event":"connect","client":"127.0.0.1","user":"julio","method":"CONNECT","dest":"example.com:443","resolved":["93.184.215.14"],"decision":"allow","reason":"permit_name","status":200,"duration_ms":5321,"bytes_up":1840,"bytes_down":9902}
{"time":"2026-11-25T23:58:09-05:00","event":"connect","client":"10.0.0.7","method":"CONNECT","dest":"blocked.example:443","decision":"deny","reason":"block_name","status":403,"duration_ms":0,"bytes_up":0,"bytes_down":0}
```

| Field | Meaning |
|---|---|
| `time` | When the request arrived (RFC 3339). |
| `event` | `connect`, `http`, `websocket`, or `unknown`. |
| `client` | Client IP address. |
| `user` | Authenticated user name, when auth is enabled. |
| `method`, `dest` | HTTP method and `host:port` requested. |
| `resolved` | IP addresses the name resolved to. |
| `decision` | `allow` or `deny`. |
| `reason` | Why: `permit_name`, `permit_address`, `block_name`, `block_address`, `special_range`, `dns_failure`, `default_permit`, `default_block`, `too_many_connections`, `tls_required`, `auth_missing`, `auth_blocked`, `auth_failed`, `upstream_error`, `internal_error`, `shutting_down`, `unsupported_request`. |
| `status` | HTTP status returned to the client. |
| `duration_ms` | Time from arrival to the end of the request or tunnel. |
| `bytes_up`, `bytes_down` | Bytes sent by the client and by the destination. |

Never logged: passwords, `Proxy-Authorization` headers, request paths, query strings, or payloads. Only the host of a forwarded HTTP request is recorded.

Failed logins and blocked IPs are also logged on the `[AUTH]` lines, regardless of `access_log`.

## Metrics

Metrics are served on their own HTTP listener, separate from the proxy ports, so a collector can scrape them without touching proxy traffic. It is off unless `metrics_addr` is set:

```yaml
metrics_addr: ":9100"               # e.g. 0.0.0.0:9100 to accept scrapes from the network
metrics_allow:                       # optional: only these IPs or CIDR ranges may scrape
  - 10.0.0.0/8
```

`GET /metrics` returns counters in the Prometheus text format. With `metrics_allow` empty, any client that can reach `metrics_addr` can read them, so restrict the port with a firewall or with `metrics_allow`. Other paths return `404`.

| Metric | Labels | Meaning |
|---|---|---|
| `doppler_connections_active` | | Proxied connections in progress (gauge). |
| `doppler_requests_total` | `event` | Proxied requests: `connect`, `http`, `websocket`, `unknown`. |
| `doppler_decisions_total` | `decision`, `reason` | Allow and deny decisions, with the reason (see the access log reasons). DNS failures are `reason="dns_failure"`. |
| `doppler_auth_failures_total` | `reason` | Failed authentication: `auth_missing`, `auth_blocked`, `auth_failed`, `auth_requires_tls`. |
| `doppler_connection_errors_total` | `reason` | Errors connecting upstream or internally. |
| `doppler_bytes_total` | `direction` | Bytes transferred, `up` (client to destination) or `down`. |

Metrics are kept in memory and reset when the service restarts. They are updated even when `access_log` is off.

## Resource limits

| Option | Default | Effect |
|---|---|---|
| `max_connections` | 1024 | Concurrent proxied connections (tunnels and forwarded requests). Extra requests get `503`. `0` = no limit. |
| `idle_timeout` | 300 s | Tunnels with no traffic for this long are closed. Also limits idle keep-alive connections on the HTTP server. `0` = never. |
| `max_tunnel_seconds` | 0 | Closes a tunnel after this long, even if it is active. `0` = no limit. |
| `limit` | 5000 KB/s | Throughput per direction for tunnels. |
| `max_header_bytes` | 16384 | Maximum request header size. `0` = Go default (1 MB). |
| `dial_timeout` | 10 s | How long to wait for a destination to accept the connection. |
| `auth.max_failures` and related | 5 / 15 min | Failed logins per IP before a temporary block. |

The service unit sets `LimitNOFILE=8192`, because every tunnel uses two sockets.

## Configuration

### Validation

At startup Doppler checks the whole configuration and refuses to start if anything is wrong, listing every problem:

```
2026/11/26 00:15:05 invalid configuration:
  - addr: "5000" is not host:port
  - ACL.default: must be "permit" or "block" (got "allow")
  - ACL.permit: "10.0.0.0/33" is not a valid CIDR range
```

It checks addresses and ports, `ACL.default`, every permit and block entry (IP, CIDR, name or leading `*.` wildcard), non-negative numbers, `metrics_allow`, TLS files, and, when auth is on, that TLS is enabled, users exist, and every password is a bcrypt hash.

### Reload

Send `SIGHUP` to reload the file without restarting:

```bash
sudo systemctl kill -s HUP doppler
```

The new file is validated first. If it is invalid, Doppler logs the reason and keeps the running configuration. If it is valid, it takes effect for the next request or connection: ACL, auth users and limits, access logging, and `metrics_allow`.

Some settings are read only when the listeners start: `addr`, `tls_addr`, `tls_cert`, `tls_key`, `metrics_addr`, `max_connections`, `read_timeout`, `write_timeout`, and `max_header_bytes`. If a reload changes one of them, Doppler logs that it needs a restart and keeps the running value. Restart the service to apply it.

## Testing

```bash
go test ./...                 # all tests, about 20 seconds
go test -race ./...           # same, with the race detector (what CI runs)
go test ./handler -run='^$' -fuzz=FuzzParseBasic -fuzztime=60s   # longer fuzzing
```

The tests run the real proxy in memory against local servers, and the service binary for startup, reload and shutdown. They do not depend on the service, the public internet, or the machine's DNS (names are resolved by a fixed table where needed). CI (`.github/workflows/ci.yml`) runs format, vet, race-enabled tests and a build on every push and pull request.

| Area | Covered by |
|---|---|
| Domains: permit, block, both lists, default allow/deny, case, trailing dot | `handler/proxy_test.go`, `internal/utils/acl_test.go` |
| Wildcards `*.example.com` and apex behavior | `handler/suite_test.go`, `internal/utils/acl_test.go` |
| Internationalized names (punycode) | `handler/suite_test.go`, `internal/utils/acl_test.go` |
| Multiple resolved IPs, DNS failure | `handler/proxy_test.go` |
| Direct IP, IPv4, IPv6, IPv4-mapped, CIDR, special ranges | `handler/proxy_test.go`, `internal/utils/acl_test.go` |
| Authentication: success, failure, over TLS only, rate limit | `handler/proxy_test.go`, `handler/auth_test.go` |
| Invalid and malformed CONNECT | `handler/proxy_test.go`, `handler/suite_test.go` |
| WebSocket upgrade and policy | `handler/websocket_test.go` |
| Client and destination disconnects, half-close, goroutine cleanup | `handler/lifecycle_test.go`, `internal/utils/relay_test.go` |
| Destination timeout (`dial_timeout`) | `handler/suite_test.go` |
| Concurrent tunnels, pipelined data | `handler/proxy_test.go`, `handler/lifecycle_test.go` |
| Limits: connections, tunnel duration, idle timeout, header size | `handler/lifecycle_test.go`, `internal/utils/relay_test.go`, `server/server_test.go` |
| PAC endpoint: TLS and loopback rules, public_host, missing file | `server/server_test.go` |
| Access log fields and secrets | `handler/access_test.go` |
| Metrics, allow list, separate listener | `internal/utils/metrics_test.go`, `handler/metrics_test.go`, `server/server_test.go` |
| Config validation and reload | `config/config_test.go` |
| Startup rejection, SIGHUP reload, SIGTERM shutdown (real binary) | `main_test.go` |
| Fuzzing: auth header, ACL entries, YAML config, name and IP matching | `handler/fuzz_test.go`, `config/fuzz_test.go`, `internal/utils/fuzz_test.go` |

The IPv6 destination test is skipped when the machine has no IPv6 loopback.

## Installation and Usage

### Requirements

- Go 1.26 or later (see `go.mod`).
- Linux with systemd for the service installer.

### Run locally (development)

```bash
./run.sh
```

Builds the binary into `dist/`, creates a self-signed certificate in `certs/` if missing, and starts Doppler with HTTP on `:5080` and TLS on `:5443`.

### Install as a service

```bash
sudo ./install.sh
```

Builds the binary and installs it to `/usr/bin/doppler`. It creates `/etc/doppler/` with the configuration, a self-signed certificate in `/etc/doppler/certs/` (if missing), and `proxy.pac`. It creates the `doppler` system user, installs `doppler.service`, and starts it.

Existing values in `/etc/doppler/config.yml` are kept, but note that the installer copies the repository's `config.yml` over it on each run.

### Remove

```bash
sudo ./uninstall.sh          # stops the service, keeps /etc/doppler
sudo ./uninstall.sh --purge  # also removes /etc/doppler and the doppler user
```

### Configure

Copy `config.yml.example` to `config.yml` and edit it. Every option is documented in the example and in the sections above. The most common ones:

```yaml
addr: ":5000"              # plain HTTP listener
tls_addr: ":5443"          # TLS listener, needed for auth and for the PAC
tls_cert: "/etc/doppler/certs/tls.crt"
tls_key: "/etc/doppler/certs/tls.key"
public_host: "proxy.example.com"   # host written into the PAC file

ACL:
  default: block
  permit:
    - "*.example.com"
    - 10.0.0.0/8
  block:
    - "*.ads.example"

auth:
  enabled: true
  users:
    julio: "<bcrypt hash>"   # go run ./cmd/hashpw

metrics_addr: ":9100"
metrics_allow:
  - 10.0.0.0/8
```

Changes are checked at startup, and most of them take effect on reload: `sudo systemctl kill -s HUP doppler`. Settings that need a restart are listed in "Configuration".

### Create a user

```bash
read -rs P && printf '%s\n' "$P" | go run ./cmd/hashpw -c /etc/doppler/config.yml; unset P
```

Paste the printed hash into `auth.users` and reload. The password must meet `auth.password` (length, upper case, digit, symbol by default).

### Use it in a browser

Point the browser's automatic proxy configuration at `https://<host>:5443/proxy.pac`, using the TLS certificate you configured. On the same machine, `http://127.0.0.1:5000/proxy.pac` also works.

### Checks

Run `go test ./...` before deploying. CI (`.github/workflows/ci.yml`) runs the same checks on every push.

## Transparent Traffic Redirection

Doppler can be combined with tools such as [redsocks](https://github.com/darkk/redsocks) to redirect compatible outbound traffic transparently through the proxy.

This allows applications to use the proxy without requiring individual proxy configuration.

A possible architecture is:

```text
Applications
     |
     v
Traffic Redirection
     |
     v
  Redsocks
     |
     v
Doppler Proxy
     |
     v
External Destinations
```

Transparent redirection is provided by the surrounding network configuration and tools such as redsocks, not by Doppler itself.

## Technology Stack

- **Language:** Go 1.26.
- **Proxy protocols:** HTTP CONNECT, HTTP forwarding, WebSocket upgrade.
- **Configuration:** YAML, validated at startup.
- **Dependencies:** `gorilla/mux` (routing), `golang.org/x/crypto/bcrypt` (password hashes), `golang.org/x/net/idna` (internationalized names), `gopkg.in/yaml.v2`.
- **Observability:** JSON access log on stdout, Prometheus text metrics.

## Security Considerations

Doppler includes authentication and destination policies. They are not a complete security boundary.

- **Credentials travel only over TLS.** With `auth.enabled: true`, the plain listener refuses proxy requests, and the service does not start without `tls_addr`. Clients must trust the certificate (`tls_cert`). The installer generates a self-signed certificate for `localhost` and `127.0.0.1`; for any other host, use a certificate issued for that name.
- **The plain listener is still open** when auth is off. Keep it on a trusted network or behind a firewall.
- **The PAC file** decides where all browser traffic goes. It is served over HTTP only to clients on this machine; everything else must use the TLS port.
- **Metrics** reveal counts and reasons, not destinations or credentials. Restrict the metrics port with `metrics_allow` and a firewall.
- **Passwords** are stored as bcrypt hashes. Create them with `cmd/hashpw`, which enforces the password policy.
- **Policies are not a sandbox.** Domain and IP rules depend on DNS answers at request time, and they cannot stop traffic that bypasses the proxy. Test the policy against the destinations you care about.

Doppler is an evolving engineering project. Validate it in your environment before relying on it for security-critical traffic.

## Current Status

Doppler is a command-line proxy configured through YAML. It runs as a systemd service on Linux, with an installer and an uninstaller.

It has no graphical interface and no administrative API. Its management is done through the configuration file, `SIGHUP` reloads, and the access log and metrics endpoints.

## Future Improvements

- Administrative API for policy and user management.
- Optional web-based management interface.
- Per-user policies (currently the policy is global; authentication identifies the user in logs and metrics only).
- Connection-level metrics that also count active tunnels by reason.
- Automated installer tests and a test for the uninstaller.

## Origin

Doppler originated from an engineering idea inspired by a banking software project: what if outbound network requests could be centrally controlled, with everything denied unless explicitly authorized?

The initial implementation evolved into a personal proxy that could also route internet traffic through remote infrastructure, including AWS.

Today, Doppler remains an independent project exploring practical approaches to proxying and outbound network access control.

## License

See the repository's LICENSE file for licensing information.