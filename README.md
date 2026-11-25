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

Doppler supports three main policy components:

- `permit`: Destinations explicitly allowed.
- `block`: Destinations explicitly denied.
- `defaultPolicy`: The decision applied when no explicit rule matches.

For restrictive environments, a default-deny policy is the recommended starting point.

Conceptually:

```yaml
defaultPolicy: block

permit:
  - example.com
  - api.example.com

block:
  - blocked.example.com
```

The exact configuration structure should follow the implementation's current configuration schema.

### Policy Evaluation

Doppler evaluates both the requested hostname and the IP addresses obtained through DNS resolution.

The current evaluation behavior gives block rules priority over permit rules when the same value matches both lists.

Unmatched destinations are evaluated using the configured default policy.

**Important:** Domain and IP-based access policies have limitations that should be considered when designing security-sensitive deployments. DNS resolution behavior, multiple resolved addresses, and IP-based access should be tested carefully.

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
- Test DNS resolution and access-control behavior.
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