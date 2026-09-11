# Contributing

Please refer to Cloudsmith's standard guide on [Open-Source Contributing](https://docs.cloudsmith.com/resources/contributing).


## Contributor License Agreement

By making any contributions to Cloudsmith Ltd projects you agree to be bound by the terms of the Cloudsmith Ltd [Contributor License Agreement](https://docs.cloudsmith.com/resources/contributor-license-agreement).


## Development Environment

The basic requirements are:

- [Go](https://go.dev/dl/), at the version `go.mod` names. The test harness is a Go module.
- Docker, for the container tests.
- The [`sbx` CLI](https://github.com/docker/sbx-releases), for spec validation and the end-to-end tests.

Kits are experimental. The installer defaults to `sbx v0.42.1`; pass `latest`
explicitly when checking compatibility with a newer release. It verifies the
download against GitHub's release-asset SHA256 digest. On macOS ARM64,
`CLI_ONLY=1 PREFIX="$PWD/.tools/sbx" ./scripts/install-sbx.sh` installs the
daemon-free CLI locally; a full sandbox installation is still needed for e2e.


## Before Opening a Pull Request

Run the offline regression suite, then these for every kit you touch:

```console
go test -race ./...               # kit contracts and mocked setup failures
go vet ./...
shellcheck scripts/*.sh
./scripts/test-kit.sh <kit>        # container TCK
./scripts/test-kit-e2e.sh <kit>    # real sandbox under deny-all
sbx kit validate ./<kit>
```

`scripts/test-kit-e2e.sh` needs a one-time sign-in. The scoped daemon keeps its
own credential store, separate from your day-to-day sbx:

```console
sbx --app-name cs-kits login
```

The e2e run creates a real sandbox under `deny-all` and tests install-time
network access and applied configuration. Neither the container TCK nor the
generic e2e harness proves that every package manager, private credential,
redirect, or firewall denial works. Test those operations explicitly before
release, using least-privilege test credentials.

The wrapper exports `APP_NAME` to the Go harness. The default is `cs-kits`;
isolated overrides must match `cs-kits-<1-8 lowercase letters/digits>`.
Sign in to the same app name. The wrapper resets only that daemon's policy;
never use it for a daemon containing day-to-day sandboxes.

Test the composed configurations too, not only the kit you changed:

```console
sbx --app-name cs-kits run claude --kit ./cloudsmith-repo --kit ./cloudsmith-dependency-firewall --kit-arg cloudsmith-repo.path=/ORG/REPO .
```


## What CI Covers

`.github/workflows/tck.yml` runs on every pull request and push to `main`:

- **`tck`** — the container TCK for each kit, through the published
  `github.com/docker/sbx-kits-contrib` module pinned in `go.mod`.
- **`regression`** — offline kit contracts, default/custom-domain rendering,
  argument rejection, mocked HTTP and CLI installer failures/cleanup, Go
  race/vet checks and ShellCheck.
- **`validate`** — `sbx kit validate` and `sbx kit inspect` against the pinned
  CLI release.
- **`e2e`** — the real-sandbox test, through `.github/workflows/e2e.yml`, only
  on pushes to `main`. Pull-request code never receives Docker Hub credentials.

The end-to-end job needs a Docker Hub credential to pull the sandbox template.
Set the `DOCKERHUB_USERNAME` repository variable and the `DOCKERHUB_TOKEN`
repository secret to turn it on. Without them the job **skips with a notice**
and the build stays green. A green build is therefore not proof that the
end-to-end test ran. Check the job, and run it locally before you merge.

All pull requests skip e2e, including same-repository pull requests.


## Updating the Test Harness

Bump the pinned TCK:

```console
go get github.com/docker/sbx-kits-contrib@v0.19.1
go mod tidy
go test -race ./...
./scripts/test-kit.sh cloudsmith-repo
```

The published TCK is currently pinned to `v0.19.1`. For an upgrade, replace that
version deliberately and validate all kits against both the module and CLI.
Do not vendor or rebuild the harness merely because the CLI releases separately.


## Per-Kit README

Every kit ships a `README.md` covering what it does, its arguments, the `sbx
run` invocation, its network allowlist and the reason for each entry, and the
non-obvious decisions in the spec. A reviewer must not have to reverse-engineer
the YAML.


## Network Policy

`permissions.network.allow` lists the access a kit requests; it is not a
standalone outbound boundary. Agent kits and daemon/organization policy also
participate, and deny rules win. Every domain
a credential injects into (`credentials[].apiKey.inject[].domain`) must also
appear there. The engine does not check this, so get it right in review. The
end-to-end run under `deny-all` checks only the endpoints the tests exercise.


## Releasing

Each kit is tagged on its own: `<kit>-v<semver>`, for example
`cloudsmith-repo-v0.1.0`. A consumer can pin two kits at different points, so
keep the tags independent even when one pull request changes both.

1. Bump `version:` in the kit's `spec.yaml`.
2. Merge to `main`.
3. Run the `Sign kits` workflow (`gh workflow run sign.yml`). It signs every
   kit keyless through Sigstore, verifies each signature, and opens a pull
   request with the `kit.sig.bundle` sidecars. Signing must come before the
   tag: the signature covers the exact bytes, so any later edit invalidates it.
4. Review and squash-merge that pull request. GitHub signs the squash commit.
   Workflow-created pull requests do not trigger normal push/PR workflows with
   `GITHUB_TOKEN`; run the required checks explicitly or use an approved bot
   workflow before merging. Repository Actions settings must permit creation
   of pull requests.
5. Tag the squashed commit `<kit>-v<semver>` and push the tag.

The keyless signing workflow currently uploads to the public Sigstore
transparency log. For a private repository, approve that disclosure before
running it: certificate identity and signing metadata become public.

Verify a Git-distributed kit's committed bundle after signing:

```console
sbx kit verify \
  --certificate-identity https://github.com/cloudsmith-io/cloudsmith-sbx-kits/.github/workflows/sign.yml@refs/heads/main \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com \
  ./cloudsmith-repo
```

Pushing the tag starts `.github/workflows/publish.yml`, which publishes the kit
to Cloudsmith as an OCI artifact at
`docker.cloudsmith.io/<namespace>/<repo>/<kit>-kit:v<semver>`. The workflow
authenticates with OIDC, so no API key is stored. It needs three repository
variables, and fails closed if any are missing:

| Variable | Value |
| --- | --- |
| `CLOUDSMITH_NAMESPACE` | the Cloudsmith workspace |
| `CLOUDSMITH_REPO` | the repository holding the kit artifacts |
| `CLOUDSMITH_SERVICE_SLUG` | a service account whose OIDC provider trusts only this repository's intended release-tag subjects and audience, not pull requests or arbitrary branches |

The workflow checks out the exact tag (including manual dispatch), requires it
to be an ancestor of `main`, validates its kit name/version, and verifies the
committed bundle against the exact `sign.yml@refs/heads/main` identity. Signing
is allowed only from `main`. Existing OCI versions are rejected rather than
assumed to contain the requested bytes; cut a new version. Registry authentication
and transport failures abort instead of being treated as an absent artifact.

**The OCI artifact and its provenance remain unsigned.** A committed
`kit.sig.bundle` verifies Git-distributed kit content, not a Git tag or an OCI
manifest. Current Docker documentation supports `sbx kit push --sign` and
Docker-credential-store fallback for non-Hub registries, but this repository
has not verified signed Cloudsmith publishing and referrer retrieval. Validate
that flow before enabling it; an unsigned OCI artifact cannot satisfy
`kit.requireSignature=true`. No existing repository settings or OIDC trust are
assumed to be configured by these workflows.


## Need Help?

See the section for raising a question in the [Contributing Guide](https://docs.cloudsmith.com/resources/contributing).
