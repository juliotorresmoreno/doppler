# Doppler — Technical Review & Improvement Report

**Project:** Doppler HTTP CONNECT Proxy  
**Language:** Go  
**Review status:** Preliminary  
**Objective:** Identify security, correctness, reliability, and product improvements before expanding Doppler's capabilities.

---

## 1. Critical: Security

### 1.1 TLS for client-to-proxy communication

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

**Current situation**

The access-control function resolves the requested hostname and evaluates the resulting IP addresses.

A potential problem exists if DNS resolution fails while the default policy is `permit`.

**Proposed improvement**

Define explicit behavior for DNS resolution failures.

For restrictive environments, DNS resolution failures should result in denied connections.

**Priority:** Critical.

### 2.2 Multiple resolved IP addresses

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

**Things to review**

Determine whether clients can issue CONNECT requests using literal IP addresses instead of hostnames.

If so, ensure that these requests are evaluated against the same access-control policies.

**Priority:** High.

---

## 3. Proxy Reliability

### 3.1 Connection lifecycle

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

Review existing transfer limits and determine whether additional controls are necessary:

- Maximum concurrent connections.
- Maximum connection duration.
- Idle connection timeout.
- Maximum request header size.
- Maximum authentication attempts.
- Memory and file descriptor usage.

**Priority:** Medium.

### 3.3 Connection reuse

Investigate whether Doppler creates independent TCP connections for each CONNECT request and whether any connection reuse exists.

HTTP CONNECT normally establishes a dedicated tunnel, so connection pooling should not be assumed to be beneficial or appropriate.

**Priority:** Low / Investigation.

---

## 4. Observability and Administration

### 4.1 Access logs

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

Investigate whether configuration changes require restarting the proxy.

Potential improvements:

- Configuration validation.
- Better startup error messages.
- Safe configuration reload.
- Atomic policy updates.

**Priority:** Medium.

---

## 5. Testing

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

| Priority | Task | Category |
|---|---|---|
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