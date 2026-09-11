# cloudsmith-sbx-kits

[Docker Sandboxes](https://docs.docker.com/ai/sandboxes/) kits that make
[Cloudsmith](https://cloudsmith.com) the package source for a coding agent's
sandbox. The agent installs dependencies through your repository, so your
vulnerability, license and deny policies decide what it gets. No credential
ever reaches the agent: the sandbox proxy adds authentication on the way out.

```
cloudsmith-repo/                  point pip/uv, npm, Go, cargo, Maven, NuGet, Docker and raw at a Cloudsmith repository
cloudsmith-cli/                   install the cloudsmith CLI, with a separate API key
cloudsmith-dependency-firewall/   deny the public registries so nothing can fall back to them
```

Each kit is a `kind: mixin` on `schemaVersion: "2"`. They work with any base
agent and compose in any combination.

## Quick start

Allow this repository as a kit source once. `sbx` trusts only `docker.io/` by
default. Keep any sources you already allow:

```console
sbx settings set kit.allowedSources '["docker.io/","github.com/cloudsmith-io/"]'
```

Store the repository's entitlement token. Skip this for a public repository:

```console
sbx secret set cloudsmith-entitlement-token
```

Run an agent whose packages come from your repository, with the public
registries denied:

```console
sbx run claude \
  --kit "git+https://github.com/cloudsmith-io/cloudsmith-sbx-kits.git#ref=cloudsmith-repo-v0.1.0&dir=cloudsmith-repo" \
  --kit "git+https://github.com/cloudsmith-io/cloudsmith-sbx-kits.git#ref=cloudsmith-dependency-firewall-v0.1.0&dir=cloudsmith-dependency-firewall" \
  --kit-arg cloudsmith-repo.path=/your-org/your-repo .
```

Approve the credential binding on first use. Add `--kit ...&dir=cloudsmith-cli`
and store `cloudsmith-api-key` when the agent also needs the Cloudsmith API.

Or clone and reference by path, which is easier while you try changes:

```console
git clone https://github.com/cloudsmith-io/cloudsmith-sbx-kits.git
sbx run claude --kit ./cloudsmith-sbx-kits/cloudsmith-repo --kit-arg cloudsmith-repo.path=/your-org/your-repo .
```

**Pin every remote reference.** A `git+https://` reference with no `#ref=`
resolves to whatever `main` holds when the kit is pulled. Use a tag, or a
40-character commit SHA for the strictest guarantee.

## Which kits to compose

| | `cloudsmith-repo` | `+ cloudsmith-dependency-firewall` | `+ cloudsmith-cli` |
| --- | --- | --- | --- |
| Packages come from Cloudsmith | yes | yes | yes |
| Public registries reachable | yes | **no** | unchanged |
| Credential carried | read-only entitlement token | none | API key, separate |
| Cloudsmith API reachable | no | no | yes |

`cloudsmith-repo` is a redirect. It points every package client at your
repository, but a project-level `.npmrc` or an explicit `--index-url` can still
route around it. `cloudsmith-dependency-firewall` closes that path: a fallback
to PyPI, npm, crates.io, Maven Central, nuget.org or Docker Hub becomes a
policy block you can see in `sbx policy log`. Adopt the redirect first, then
add enforcement with one more `--kit`.

Keep `cloudsmith-cli` out unless the agent needs the API. Its key carries
whatever its owner can do, including publish and delete. The entitlement token
in `cloudsmith-repo` is read-only and scoped to one repository.

Read each kit's own README for its arguments, supported formats and limits:
[`cloudsmith-repo`](./cloudsmith-repo/README.md),
[`cloudsmith-cli`](./cloudsmith-cli/README.md),
[`cloudsmith-dependency-firewall`](./cloudsmith-dependency-firewall/README.md).

## Verify a kit

Every released kit is signed keyless through Sigstore, bound to this
repository's `sign.yml` workflow identity. Verify before you run:

```console
sbx kit verify \
  --certificate-identity-regexp "^https://github.com/cloudsmith-io/cloudsmith-sbx-kits/" \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com \
  "git+https://github.com/cloudsmith-io/cloudsmith-sbx-kits.git#ref=cloudsmith-repo-v0.1.0&dir=cloudsmith-repo"
```

## Versioning

Each kit carries its own `version:` and its own tag, `<kit>-v<semver>` — for
example `cloudsmith-repo-v0.1.0`. The kits compose independently, so you can
pin them at different points. Tags do not move.

## Development

```console
./scripts/test-kit.sh cloudsmith-repo        # container TCK, needs Docker
./scripts/test-kit-e2e.sh cloudsmith-repo    # real sandbox under deny-all, needs sbx
sbx kit validate ./cloudsmith-repo
```

The TCK comes from the published `github.com/docker/sbx-kits-contrib` module,
pinned in `go.mod`. These kits are tested by the same harness Docker's own kit
repository uses, rather than by a copy of it that drifts. See
[CONTRIBUTING.md](./CONTRIBUTING.md).

## Limits

These are real, and stated here rather than discovered later:

1. **Kits are experimental.** The spec, the CLI and the kit management
   experience change. These kits are verified against sbx 0.42.x.
2. **`sbx` network policy is per hostname.** The firewall kit denies the
   registries it names. It cannot express "nothing except this one
   repository", and it does not restrict a shared host such as
   `dl.cloudsmith.io` to a single Cloudsmith repository. A custom download
   domain narrows that to one organization.
3. **One Cloudsmith endpoint can leak the entitlement token.** A raw
   `versions/latest/<file>` request answers with a redirect whose `Location`
   carries the token. Pin raw downloads to a version. Treat the token as
   recoverable by hostile code in the sandbox, and rotate it after such a run.
4. **The allowlist is a strong default, not an unbreakable control.** A
   developer can widen it with `sbx policy allow`. Audit with `sbx policy log`.
5. **The agent instructions are guidance, not enforcement.** They tell the
   agent to report a failed download instead of switching to a public
   registry. The firewall kit is what makes that a control.
6. **`cloudsmith-repo` replaces three home config files** on every start:
   `~/.cargo/config.toml`, `~/.m2/settings.xml` and
   `~/.nuget/NuGet/NuGet.Config`.

## License

Apache-2.0. See [LICENSE](./LICENSE).
