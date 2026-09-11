# Cloudsmith sandbox kits

[Docker Sandboxes](https://docs.docker.com/ai/sandboxes/) mixins for installing
dependencies through [Cloudsmith](https://cloudsmith.com), accessing its API,
and blocking named public package registries.

| Kit | Purpose | Credential |
| --- | --- | --- |
| [`cloudsmith-repo`](./cloudsmith-repo/README.md) | Configure pip/uv, npm/pnpm/Yarn, Go, Cargo, Maven and NuGet; enable explicit Cloudsmith Docker and raw URLs | Optional read-only entitlement token |
| [`cloudsmith-cli`](./cloudsmith-cli/README.md) | Install the Cloudsmith CLI and enable API access | Optional API key |
| [`cloudsmith-dependency-firewall`](./cloudsmith-dependency-firewall/README.md) | Deny the listed PyPI, npm, Go, crates.io, Maven Central, NuGet and Docker Hub endpoints | None |

> [!IMPORTANT]
> Kits are experimental. This repository targets sbx 0.42.x, with 0.42.1 as
> the compatibility baseline. The host proxy injects credentials, and client
> configuration does not store them. This is **not a guarantee that a hostile
> agent cannot recover them**: Cloudsmith raw `versions/latest/`
> redirects can expose the entitlement token. See [Security](./SECURITY.md).

## Quick start

Install [Docker Sandboxes](https://docs.docker.com/ai/sandboxes/) and configure
authentication for your chosen base agent. Docker Engine alone cannot replace
the `sbx` runtime.

To try the kits from a local checkout:

```console
git clone https://github.com/cloudsmith-io/cloudsmith-sbx-kits.git
cd cloudsmith-sbx-kits
```

For a private repository, store a read-only, repository-scoped
[entitlement token](https://docs.cloudsmith.com/software-distribution/entitlement-tokens)
on the host. Skip this for public repositories:

```console
sbx secret set cloudsmith-entitlement-token
```

Create a sandbox with your project mounted and public-registry blocking enabled:

```console
sbx run claude \
  --kit ./cloudsmith-repo/ \
  --kit ./cloudsmith-dependency-firewall/ \
  --kit-arg cloudsmith-repo.path=/your-org/your-repo /absolute/path/to/your-project
```

Replace the repository and project paths. `kit.allowLocalKits` must permit local
kits, which is the default. The repository needs upstreams for packages not
already uploaded to it. Check the
[repository kit prerequisites](./cloudsmith-repo/README.md#usage) before using a
custom base image.

If the organization has a custom download domain, also pass
`--kit-arg cloudsmith-repo.redirect-host=dl.your-domain.example` and approve
that host in the credential binding. See
[custom-domain configuration](./cloudsmith-repo/README.md#repositories-with-custom-download-domains).

Add `--kit ./cloudsmith-cli/` and store `cloudsmith-api-key` with
`sbx secret set` when the agent also needs API access. That key is independent
of the entitlement token and can grant write or delete access.

Every kit takes effect at sandbox creation. Recreate an existing sandbox to add
one. Under organization governance, ask your administrator to allow the hosts
each kit lists; kit allow rules cannot grant that access.

### Credential bindings

On first use, approve the kit's hosts in the credential-binding prompt. For
unattended runs, create the binding beforehand by running interactively once or
by configuring
[credential bindings](https://docs.docker.com/ai/sandboxes/configuration/credentials/#credential-bindings).
Without an approved binding, the proxy withholds the credential and
authenticated requests fail.

Remove a stored secret with `sbx secret rm <name> -f`, and only when no other
sandbox needs it. That does not revoke the credential in Cloudsmith. See
[Rotating a Credential](./SECURITY.md#rotating-a-credential).

## Use a released kit

You can load released kits from signed Git references or signed Cloudsmith OCI
artifacts. OCI avoids cloning this repository; Git lets you select a source
revision directly. See [Use Cloudsmith OCI artifacts](#use-cloudsmith-oci-artifacts)
for registry references and verification.

Allow this repository as a remote source. The following setting replaces the
whole list: retain any other sources you already trust.

```console
sbx settings set kit.allowedSources '["docker.io/","github.com/cloudsmith-io/cloudsmith-sbx-kits"]'
```

Each kit has an independent `version:` and immutable release tag,
`<kit>-v<version>`. Check [Tags](https://github.com/cloudsmith-io/cloudsmith-sbx-kits/tags)
for available versions. The following uses the published `v0.1.0` kits:

```console
sbx run claude \
  --kit "git+https://github.com/cloudsmith-io/cloudsmith-sbx-kits.git#ref=cloudsmith-repo-v0.1.0&dir=cloudsmith-repo" \
  --kit "git+https://github.com/cloudsmith-io/cloudsmith-sbx-kits.git#ref=cloudsmith-dependency-firewall-v0.1.0&dir=cloudsmith-dependency-firewall" \
  --kit-arg cloudsmith-repo.path=/your-org/your-repo /absolute/path/to/your-project
```

Quote Git URLs containing `&`. Pin to a release tag or a full commit SHA;
without `ref`, the kit tracks the repository's default branch.

### Verify signatures

Git kits need a committed `kit.sig.bundle` at the referenced revision. A
version or tag alone does not prove a kit is signed. For a release signed by
this repository's `sign.yml` workflow on `main`:

```console
sbx kit verify \
  --certificate-identity https://github.com/cloudsmith-io/cloudsmith-sbx-kits/.github/workflows/sign.yml@refs/heads/main \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com \
  "git+https://github.com/cloudsmith-io/cloudsmith-sbx-kits.git#ref=cloudsmith-repo-v0.1.0&dir=cloudsmith-repo"
```

Verification covers the spec and bundled files, not everything an install
command downloads later. See
[Docker's signing documentation](https://docs.docker.com/ai/sandboxes/customize/kits/#sign-and-verify-kits).

### Use Cloudsmith OCI artifacts

The public kit artifacts are available without Cloudsmith credentials:

| Kit | OCI reference |
| --- | --- |
| Repository | `docker.cloudsmith.io/cloudsmith/sbx-kits/cloudsmith-repo-kit:v0.1.0` |
| CLI | `docker.cloudsmith.io/cloudsmith/sbx-kits/cloudsmith-cli-kit:v0.1.0` |
| Dependency firewall | `docker.cloudsmith.io/cloudsmith/sbx-kits/cloudsmith-dependency-firewall-kit:v0.1.0` |

These are sandbox kit artifacts, not runnable container images. Use them with
`sbx --kit`, not `docker run`. Allow their source, retaining any other sources
you already trust:

```console
sbx settings set kit.allowedSources '["docker.io/","github.com/cloudsmith-io/cloudsmith-sbx-kits","docker.cloudsmith.io/cloudsmith/sbx-kits/"]'

sbx run claude \
  --kit docker.cloudsmith.io/cloudsmith/sbx-kits/cloudsmith-repo-kit:v0.1.0 \
  --kit docker.cloudsmith.io/cloudsmith/sbx-kits/cloudsmith-dependency-firewall-kit:v0.1.0 \
  --kit-arg cloudsmith-repo.path=/your-org/your-repo /absolute/path/to/your-project
```

The entitlement token and credential-binding steps in [Quick start](#quick-start)
still apply when the package repository you select is private. Fetching a
public kit and accessing its configured package repository are separate operations.

The published `v0.1.0` artifacts carry the `publish.yml@refs/heads/main`
identity. Verify it before you use them:

```console
sbx kit verify \
  --certificate-identity https://github.com/cloudsmith-io/cloudsmith-sbx-kits/.github/workflows/publish.yml@refs/heads/main \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com \
  docker.cloudsmith.io/cloudsmith/sbx-kits/cloudsmith-repo-kit:v0.1.0
```

Repeat verification with the CLI and dependency-firewall references when using
those kits. You can pin an OCI reference by manifest digest (`@sha256:...`)
instead of a tag for content immutability, but a digest does not establish
publisher identity. Docker Hub artifacts are not available yet. Use Git or
Cloudsmith.

## Composition and boundaries

`cloudsmith-repo` sets client defaults. It does not rewrite Docker image
names, raw URLs, lockfiles or project configuration. Use Cloudsmith-qualified
image names and version-pinned raw URLs in your commands and configuration.

Add `cloudsmith-dependency-firewall` to block fallback to its named public
registry hosts. Deny rules override kit allow rules, including during
installation. The firewall is not an exclusive Cloudsmith allowlist. See
[What it does not block](./cloudsmith-dependency-firewall/README.md#what-it-does-not-block)
for the paths that stay open.

List the repository kit before kits whose installs use its package clients.
Kit deny rules apply under organization governance too.

Compose `cloudsmith-cli` only when the agent needs API access. Every process in
the sandbox can exercise the bound key's permissions, not only the CLI.

## Development and verification

```console
go test -short ./...
./scripts/test-kit.sh cloudsmith-repo
./scripts/test-kit-e2e.sh cloudsmith-repo
sbx kit validate ./cloudsmith-repo
```

Repeat the per-kit commands for `cloudsmith-cli` and
`cloudsmith-dependency-firewall`. Container tests need Docker; end-to-end
tests need `sbx` and use an isolated daemon under `deny-all`.

The upstream TCK checks the schema, installation and delivered configuration.
It does not prove authenticated downloads, quarantine behavior or all ecosystem
clients. Check private-repository behavior with scoped test credentials; the
per-kit READMEs provide smoke commands.
See [CONTRIBUTING.md](./CONTRIBUTING.md) for the full test procedure.

## Support

Report kit defects and feature requests in
[this repository's issue tracker](https://github.com/cloudsmith-io/cloudsmith-sbx-kits/issues).
Report Docker Sandboxes runtime issues to
[docker/sbx-releases](https://github.com/docker/sbx-releases/issues).
Report vulnerabilities privately as described in [SECURITY.md](./SECURITY.md).

## License

[Apache-2.0](./LICENSE). The original Cloudsmith contribution and the upstream
test harness come from
[docker/sbx-kits-contrib](https://github.com/docker/sbx-kits-contrib).
