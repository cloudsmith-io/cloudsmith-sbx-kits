# Cloudsmith sandbox kits

[Docker Sandboxes](https://docs.docker.com/ai/sandboxes/) mixins for installing
dependencies through [Cloudsmith](https://cloudsmith.com), accessing its API,
and blocking named public package registries.

| Kit | Purpose | Credential |
| --- | --- | --- |
| [`cloudsmith-repo`](./cloudsmith-repo/README.md) | Configure pip/uv, npm/pnpm/Yarn, Go, Cargo, Maven and NuGet; enable explicit Cloudsmith Docker and raw URLs | Optional read-only entitlement token |
| [`cloudsmith-cli`](./cloudsmith-cli/README.md) | Install the Cloudsmith CLI and enable API access | Optional API key |
| [`cloudsmith-dependency-firewall`](./cloudsmith-dependency-firewall/README.md) | Deny the listed PyPI, npm, Go, crates.io, Maven Central, NuGet and Docker Hub endpoints | None |

The kits retain the three-mixin design from
[docker/sbx-kits-contrib#287](https://github.com/docker/sbx-kits-contrib/pull/287).
They use schema v2 and Docker's published compatibility test harness. Agent
instructions in the specs are part of the kit's runtime behavior, not build
artifacts.

> [!IMPORTANT]
> Kits are experimental. This repository targets sbx 0.42.x, with 0.42.1 as
> the compatibility baseline. Credentials are injected by the host proxy,
> rather than stored in client configuration. This is **not a guarantee that
> a hostile agent cannot recover them**: Cloudsmith raw `versions/latest/`
> redirects can expose the entitlement token. See [Security](./SECURITY.md).

## Quick start

Install [Docker Sandboxes](https://docs.docker.com/ai/sandboxes/) and configure
authentication for your chosen base agent. Docker Engine alone can run the
container tests, but it cannot replace the `sbx` runtime.

Start from a local checkout; this also works before the first release is
published:

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

Replace the repository and project paths. Approve the credential binding on
first use. Local kits must be permitted by `kit.allowLocalKits` (the default).
The repository needs upstreams for packages not already uploaded to it.
Check the [repository kit prerequisites](./cloudsmith-repo/README.md#usage)
before using a custom base image.

If the organization has a custom download domain, also pass
`--kit-arg cloudsmith-repo.redirect-host=dl.your-domain.example` and approve
that host in the credential binding. See
[custom-domain configuration](./cloudsmith-repo/README.md#repositories-with-custom-download-domains).

Add `--kit ./cloudsmith-cli/` and store `cloudsmith-api-key` with
`sbx secret set` when the agent also needs API access. That key is independent
of the entitlement token and can grant write or delete access.

## Use a released kit

Allow this repository as a remote source. The following setting replaces the
whole list: retain any other sources you already trust.

```console
sbx settings set kit.allowedSources '["docker.io/","github.com/cloudsmith-io/cloudsmith-sbx-kits"]'
```

Each kit has an independent `version:` and immutable release tag,
`<kit>-v<version>`. Check [Releases](https://github.com/cloudsmith-io/cloudsmith-sbx-kits/releases)
for available versions before using a reference. The following example
requires both `v0.1.0` kit releases to have been published:

```console
sbx run claude \
  --kit "git+https://github.com/cloudsmith-io/cloudsmith-sbx-kits.git#ref=cloudsmith-repo-v0.1.0&dir=cloudsmith-repo" \
  --kit "git+https://github.com/cloudsmith-io/cloudsmith-sbx-kits.git#ref=cloudsmith-dependency-firewall-v0.1.0&dir=cloudsmith-dependency-firewall" \
  --kit-arg cloudsmith-repo.path=/your-org/your-repo /absolute/path/to/your-project
```

Quote Git URLs containing `&`. Pin to a release tag or a full commit SHA;
without `ref`, the kit tracks the repository's default branch.

### Verify signatures

Git kits need a committed `kit.sig.bundle` at the referenced revision.
Signing is a separate release prerequisite; the presence of a version or tag
alone does not prove a kit is signed. For a release signed by this
repository's `sign.yml` workflow on `main`:

```console
sbx kit verify \
  --certificate-identity https://github.com/cloudsmith-io/cloudsmith-sbx-kits/.github/workflows/sign.yml@refs/heads/main \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com \
  "git+https://github.com/cloudsmith-io/cloudsmith-sbx-kits.git#ref=cloudsmith-repo-v0.1.0&dir=cloudsmith-repo"
```

Verification covers the spec and bundled files, not everything subsequently
downloaded by an install command. See
[Docker's signing documentation](https://docs.docker.com/ai/sandboxes/customize/kits/#sign-and-verify-kits)
and [the release procedure](./CONTRIBUTING.md).

## Composition and boundaries

`cloudsmith-repo` sets client defaults. It does not rewrite Docker image
names, raw URLs, lockfiles or project configuration. Use Cloudsmith-qualified
image names and version-pinned raw URLs explicitly.

Add `cloudsmith-dependency-firewall` to block fallback to its named public
registry hosts. Deny rules override kit allow rules, including during
installation. The firewall is not an exclusive Cloudsmith allowlist: Git
dependencies, alternative registries and other repositories on shared hosts
may remain reachable through the composed policy. Cached packages are not
re-evaluated by a network rule.

List the repository kit before kits whose installs use its package clients.
With organization governance enabled, kit allow rules do not grant access;
the organization must permit the required Cloudsmith hosts. Kit deny rules
still apply.

Only compose `cloudsmith-cli` when API access is needed. Every process in the
sandbox can exercise the bound key's permissions, not just the CLI.

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
It does not by itself prove authenticated downloads, quarantine behavior or
all ecosystem clients. Validate private-repository behavior with scoped test
credentials before release; the per-kit READMEs provide smoke commands.
See [CONTRIBUTING.md](./CONTRIBUTING.md) for the test and release procedure.

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
