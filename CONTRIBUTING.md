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
when you check compatibility with a newer release. It verifies the
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
redirect, or firewall denial works. Test the operations your change affects
with least-privilege test credentials.

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

- **`tck`**: the container TCK for each kit, through the published
  `github.com/docker/sbx-kits-contrib` module pinned in `go.mod`.
- **`regression`**: offline kit contracts, default/custom-domain rendering,
  argument rejection, mocked HTTP and CLI installer failures/cleanup, Go
  race/vet checks and ShellCheck.
- **`validate`**: `sbx kit validate` and `sbx kit inspect` against the pinned
  CLI release.
- **`e2e`**: the real-sandbox test, through `.github/workflows/e2e.yml`, only
  on pushes to `main`. Pull-request code never receives Docker Hub credentials.

The end-to-end job needs a Docker Hub credential to pull the sandbox template.
Without one the job **skips with a notice** and the build stays green, so a
green build is not proof that the end-to-end test ran. Run the test locally
before you open the pull request.

All pull requests skip e2e, including same-repository pull requests.


## Updating the Test Harness

Bump the pinned TCK:

```console
go get github.com/docker/sbx-kits-contrib@v0.19.1
go mod tidy
go test -race ./...
./scripts/test-kit.sh cloudsmith-repo
```

`go.mod` pins the published TCK to `v0.19.1`. For an upgrade, replace that
version and validate all kits against both the module and CLI.
Do not vendor or rebuild the harness because the CLI releases separately.


## Bumping the CLI Installer

`cloudsmith-cli` pins Cloudsmith's installer script in its spec. To move to a
new installer, set `INSTALLER_VERSION` and take the matching `INSTALLER_SHA256`
from that release's `SHA256SUMS`. To move the default CLI release, change the
`cli-version` default.


## Per-Kit README

Every kit ships a `README.md` covering what it does, its arguments, the `sbx
run` invocation, its network allowlist and the reason for each entry, and the
non-obvious decisions in the spec. A reviewer must not have to reverse-engineer
the YAML.

Keep the steps that every kit shares in the repository `README.md` and link to
them. The kit source setting, release tags, signature verification, credential
bindings, the sbx version and secret removal live there once.


## Network Policy

`permissions.network.allow` lists the access a kit requests; it is not a
standalone outbound boundary. Agent kits and daemon/organization policy also
participate, and deny rules win. Every domain
a credential injects into (`credentials[].apiKey.inject[].domain`) must also
appear there. The engine does not check this, so get it right in review. The
end-to-end run under `deny-all` checks only the endpoints the tests exercise.

