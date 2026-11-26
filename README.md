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

- **HTTP CONNECT proxy:** Establish TCP tunnels to remote destinations.
- **Domain-based access control:** Allow or block connections based on requested hostnames.
- **IP-based access control:** Resolve destination hostnames and evaluate their associated IP addresses.
- **Allowlist and blocklist:** Define which destinations can or cannot be accessed.
- **Default access policy:** Configure the behavior for destinations that do not match an explicit rule.
- **Authentication:** Support authenticated proxy access.
- **Traffic limits:** Configurable transfer limits.
- **Connection timeouts:** Configurable timeout settings.
- **YAML configuration:** Manage proxy settings and access policies through configuration files.

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
- **Shuts down cleanly:** on SIGINT or SIGTERM, Doppler refuses new tunnels, closes the open ones (both sides see the close), and then stops the HTTP listeners, waiting up to 5 seconds for in-flight requests.

Destination connection attempts time out after 10 seconds.

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

- **Language:** Go
- **Proxy protocol:** HTTP CONNECT
- **Configuration:** YAML

## Security Considerations

Doppler includes proxy authentication and destination access policies. However, these features should not be interpreted as a complete security boundary.

Doppler can optionally serve the proxy over TLS on a separate port (`tls_addr`, with `tls_cert` and `tls_key`). The plain listener on `addr` remains available, so TLS is not enforced unless clients are directed to the TLS port.

Consequently, credentials and proxy requests may be exposed when the connection travels across an untrusted network.

For this reason:

- Do not expose the proxy directly to untrusted networks without additional protection.
- Use a trusted network or an encrypted transport layer.
- Review authentication and credential handling.
- Test DNS resolution and access-control behavior (see "Policy evaluation").
- Do not assume that domain filtering alone prevents every form of network access bypass.

Doppler is an evolving engineering project and should not be considered production-ready for security-critical environments without additional validation.

## Current Status

Doppler is currently a command-line-oriented proxy utility configured through YAML.

Its primary functionality is centered on authenticated HTTP CONNECT connections and domain/IP-based outbound access policies.

A graphical management interface and centralized administrative API are not currently part of the project.

## Future Improvements

Potential areas of development include:

- More comprehensive access-control tests.
- Improved DNS resolution and IP policy handling.
- Connection and access logs.
- Monitoring and operational metrics.
- Configuration reload without restarting the service.
- Administrative API.
- Optional web-based management interface.

## Origin

Doppler originated from an engineering idea inspired by a banking software project: what if outbound network requests could be centrally controlled, with everything denied unless explicitly authorized?

The initial implementation evolved into a personal proxy that could also route internet traffic through remote infrastructure, including AWS.

Today, Doppler remains an independent project exploring practical approaches to proxying and outbound network access control.

## License

See the repository's LICENSE file for licensing information.