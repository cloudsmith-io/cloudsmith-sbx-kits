# Security Policy

## Reporting a Vulnerability

Do not report security vulnerabilities through public GitHub issues.

Report them to [security@cloudsmith.io](mailto:security@cloudsmith.io), or
through Cloudsmith's [security page](https://cloudsmith.com/security). Include
the kit, the sbx version, and the steps to reproduce.

Never include a live credential in a report. Redact entitlement tokens, API
keys, pre-signed download URLs and `Location` headers.

## What These Kits Protect

The sandbox proxy holds every credential. Tools inside the sandbox see only a
`proxy-managed` placeholder, and the proxy adds the real header to requests it
forwards to the configured hosts. A hostile agent cannot read a token out of
the environment or a config file.

`permissions.network.allow` lists the hosts each kit requests.
`permissions.network.deny` beats allow rules from the base agent or another
kit. Under organization governance, organization rules must grant access;
kit allow rules alone are insufficient. End-to-end tests run under
`deny-all`, but only exercise the requests made by those tests, not every
package format, redirect or authenticated operation.

## Known Limits

Account for these limits when deciding which credentials to bind:

- **A raw `versions/latest/<file>` request can expose the entitlement token.**
  Cloudsmith answers it with a redirect whose `Location` carries the token, and
  a kit cannot block one URL on an allowed host. Pin raw downloads to a
  version. Treat the token as recoverable by hostile sandbox code, and rotate
  it after running any.
- **Network policy is per hostname.** A shared host such as `dl.cloudsmith.io`
  serves many Cloudsmith repositories, and host-level policy cannot narrow it
  to one. Custom domains can reduce the host's scope but do not replace
  repository-scoped authorization.
- **`sbx policy allow` can widen the allowlist** from the host. The allowlist
  is a strong default, not an unbreakable control. Audit with `sbx policy log`.
- **The `cloudsmith-cli` API key carries its owner's permissions**, including
  publish and delete, and every process in the sandbox can use it. Bind a
  least-privilege service-account key. Compose the kit only when the agent
  needs the API.
- **Agent instructions are guidance.** They tell the agent to report a failed
  download rather than switch to a public registry. Compose
  `cloudsmith-dependency-firewall` to enforce its named public-registry blocks.
  It does not block every alternative package source or inspect warm caches.

Kits do not replace Docker's sandbox isolation or protect files deliberately
mounted into a sandbox. Inspect kit content and credential bindings before
running it. A kit source allowlist permits downloads from that source; it is
not signature verification.

## Rotating a Credential

Revoke it in Cloudsmith first, then store the new value and recreate the
sandboxes:

```console
sbx secret set cloudsmith-entitlement-token
```

Global service secrets are reused when creating other sandboxes that bind the
same service. Removing a sandbox does not revoke its token in Cloudsmith or
remove the host secret. Do not remove a shared secret while another sandbox
still needs it.

Rotate after retiring any sandbox that ran untrusted code.
