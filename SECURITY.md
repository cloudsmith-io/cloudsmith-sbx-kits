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

`permissions.network.allow` is each kit's complete outbound contract, and
`permissions.network.deny` beats any allow from the base agent or another kit.
The end-to-end tests run under the `deny-all` policy, so an undeclared domain
fails CI rather than reaching the network.

## Known Limits

These are properties of the design, not open bugs:

- **A raw `versions/latest/<file>` request can expose the entitlement token.**
  Cloudsmith answers it with a redirect whose `Location` carries the token, and
  a kit cannot block one URL on an allowed host. Pin raw downloads to a
  version. Treat the token as recoverable by hostile sandbox code, and rotate
  it after running any.
- **Network policy is per hostname.** A shared host such as `dl.cloudsmith.io`
  serves every public Cloudsmith repository, and host-level policy cannot
  narrow it to one. A custom download domain narrows it to one organization.
- **`sbx policy allow` can widen the allowlist** from the host. The allowlist
  is a strong default, not an unbreakable control. Audit with `sbx policy log`.
- **The `cloudsmith-cli` API key carries its owner's permissions**, including
  publish and delete, and every process in the sandbox can use it. Bind a
  least-privilege service-account key. Compose the kit only when the agent
  needs the API.
- **Agent instructions are guidance.** They tell the agent to report a failed
  download rather than switch to a public registry. Compose
  `cloudsmith-dependency-firewall` to make that a control.

## Rotating a Credential

Revoke it in Cloudsmith first, then store the new value and recreate the
sandboxes:

```console
sbx secret set cloudsmith-entitlement-token
```

Rotate after retiring any sandbox that ran untrusted code.
