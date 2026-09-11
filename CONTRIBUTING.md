# Contributing

Please refer to Cloudsmith's standard guide on [Open-Source Contributing](https://docs.cloudsmith.com/resources/contributing).


## Contributor License Agreement

By making any contributions to Cloudsmith Ltd projects you agree to be bound by the terms of the Cloudsmith Ltd [Contributor License Agreement](https://docs.cloudsmith.com/resources/contributor-license-agreement).


## Development Environment

The basic requirements are:

- [Go](https://go.dev/dl/), at the version `go.mod` names. The test harness is a Go module.
- Docker, for the container tests.
- The [`sbx` CLI](https://github.com/docker/sbx-releases), for spec validation and the end-to-end tests.

Kits are experimental, and the `sbx` kit spec has already changed once inside
this project's build window. Trust a live probe against an installed `sbx`
binary over any document, including this one.


## Before Opening a Pull Request

Run these for every kit you touch:

```console
./scripts/test-kit.sh <kit>        # container TCK
./scripts/test-kit-e2e.sh <kit>    # real sandbox under deny-all
sbx kit validate ./<kit>
```

`scripts/test-kit-e2e.sh` needs a one-time sign-in. The scoped daemon keeps its
own credential store, separate from your day-to-day sbx:

```console
sbx --app-name cs-kits login
```

The end-to-end run is the one that matters. It creates a real sandbox under the
`deny-all` network policy, so it is a true test of `permissions.network.allow`.
The container TCK cannot catch a missing domain.

Test the composed configurations too, not only the kit you changed:

```console
sbx run claude --kit ./cloudsmith-repo --kit ./cloudsmith-dependency-firewall --kit-arg cloudsmith-repo.path=/ORG/REPO .
```


## What CI Covers

`.github/workflows/tck.yml` runs on every pull request:

- **`tck`** — the container TCK for each kit, through the published
  `github.com/docker/sbx-kits-contrib` module pinned in `go.mod`.
- **`validate`** — `sbx kit validate` and `sbx kit inspect` against the latest
  released `sbx` binary. The CLI and the module release separately, so this
  catches a schema change that has reached one but not the other.
- **`e2e`** — the real-sandbox test, through `.github/workflows/e2e.yml`.

The end-to-end job needs a Docker Hub credential to pull the sandbox template.
Set the `DOCKERHUB_USERNAME` repository variable and the `DOCKERHUB_TOKEN`
repository secret to turn it on. Without them the job **skips with a notice**
and the build stays green. A green build is therefore not proof that the
end-to-end test ran. Check the job, and run it locally before you merge.

Fork pull requests never receive the secret, so their end-to-end job always
skips.


## Updating the Test Harness

Bump the pinned TCK:

```console
go get github.com/docker/sbx-kits-contrib@latest
go mod tidy
./scripts/test-kit.sh cloudsmith-repo
```

Do this whenever `sbx` ships a schema change, so the harness and the CLI agree.


## Per-Kit README

Every kit ships a `README.md` covering what it does, its arguments, the `sbx
run` invocation, its network allowlist and the reason for each entry, and the
non-obvious decisions in the spec. A reviewer must not have to reverse-engineer
the YAML.


## Network Policy

`permissions.network.allow` is a kit's complete outbound contract. Every domain
a credential injects into (`credentials[].apiKey.inject[].domain`) must also
appear there. The engine does not check this, so get it right in review. The
end-to-end run under `deny-all` is what proves the list is complete.


## Releasing

Each kit is tagged on its own: `<kit>-v<semver>`, for example
`cloudsmith-repo-v0.1.0`. A consumer can pin two kits at different points, so
keep the tags independent even when one pull request changes both.

1. Bump `version:` in the kit's `spec.yaml`.
2. Merge to `main`.
3. Run the `Sign kits` workflow (`gh workflow run sign.yml`). It signs every
   kit keyless through Sigstore, verifies each signature, and commits the
   `kit.sig.bundle` sidecars. Signing must come before the tag: the signature
   covers the exact bytes, so any later edit invalidates it.
4. Tag the signing commit `<kit>-v<semver>` and push the tag.

Pushing the tag starts `.github/workflows/publish.yml`, which publishes the kit
to Cloudsmith as an OCI artifact at
`docker.cloudsmith.io/<namespace>/<repo>/<kit>-kit:v<semver>`. The workflow
authenticates with OIDC, so no API key is stored. It needs three repository
variables, and skips with a notice until all three are set:

| Variable | Value |
| --- | --- |
| `CLOUDSMITH_NAMESPACE` | the Cloudsmith workspace |
| `CLOUDSMITH_REPO` | the repository holding the kit artifacts |
| `CLOUDSMITH_SERVICE_SLUG` | a service account whose OIDC provider trusts `repo:cloudsmith-io/cloudsmith-sbx-kits:*` |

The workflow refuses a tag whose version disagrees with the kit's `spec.yaml`,
and skips a tag already published, because Cloudsmith repositories are
immutable.

**The published artifact is not signed yet.** `sbx kit push --sign` needs its
own `sbx login`, which authenticates against Docker Hub. Whether it works
against another registry is untested, so the push runs without it rather than
failing a release on an unverified flag. Signed git tags remain the verified
path — see `sign.yml`. Probe `--sign` against Cloudsmith before relying on it.


## Need Help?

See the section for raising a question in the [Contributing Guide](https://docs.cloudsmith.com/resources/contributing).
