package tck_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"
)

func publishingStep(t *testing.T, file, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("../.github/workflows", file))
	require.NoError(t, err)
	var workflow struct {
		Jobs map[string]struct {
			Steps []struct {
				Name string
				Run  string
			}
		}
	}
	require.NoError(t, yaml.Unmarshal(data, &workflow))
	var script string
	for _, step := range workflow.Jobs["publish"].Steps {
		if step.Name == name {
			script = step.Run
		}
	}
	require.NotEmpty(t, script)
	return script
}

func TestPublishingRegistryLookup(t *testing.T) {
	for _, publisher := range []struct {
		workflow string
		ref      string
	}{
		{"publish.yml", "docker.cloudsmith.io/example/kits/cloudsmith-repo-kit:v0.1.0"},
		{"publish-dockerhub.yml", "docker.io/example/cloudsmith-repo-kit:v0.1.0"},
	} {
		t.Run(publisher.workflow, func(t *testing.T) {
			for _, trigger := range []string{"refs/heads/main", "refs/tags/cloudsmith-repo-v0.1.0"} {
				t.Run(trigger, func(t *testing.T) {
					workflowRef := "example/kits/.github/workflows/" + publisher.workflow + "@" + trigger
					testPublishingRegistryLookup(t, publishingStep(t, publisher.workflow, "Push the kit artifact"), publisher.ref, workflowRef)
				})
			}
		})
	}
}

func testPublishingRegistryLookup(t *testing.T, script, ref, workflowRef string) {
	t.Helper()
	const digest = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	missing := `Error response from registry: failed to find "` + ref + `": ` + ref + `: not found`
	for _, tc := range []struct {
		name       string
		message    string
		status     string
		push       bool
		signFail   bool
		verifyFail bool
		replace    bool
	}{
		{name: "oras-not-found", message: missing, status: "1", push: true},
		{name: "manifest-unknown", message: "MANIFEST_UNKNOWN", status: "1", push: true},
		{name: "existing", status: "0"},
		{name: "unauthorized", message: "401 Unauthorized", status: "1"},
		{name: "forbidden", message: "403 Forbidden", status: "1"},
		{name: "network", message: "connection refused", status: "1"},
		{name: "unrelated-not-found", message: "credential helper: executable not found", status: "1"},
		{name: "signing-failure", message: missing, status: "1", push: true, signFail: true},
		{name: "verification-failure", message: missing, status: "1", push: true, verifyFail: true},
		{name: "authorized-replacement", status: "0", push: strings.Contains(workflowRef, "/publish.yml@"), replace: true},
		{name: "replacement-auth-failure", message: "401 Unauthorized", status: "1", replace: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			require.NoError(t, os.Mkdir(filepath.Join(dir, ".release-tools"), 0o700))
			for name, body := range map[string]string{
				"oras": `#!/bin/sh
if [ -f "$PUSHED" ] || [ "$LOOKUP_STATUS" = 0 ]; then
  printf '{"digest":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}\n'
  exit 0
fi
printf '%s\n' "$LOOKUP_ERROR" >&2
exit "$LOOKUP_STATUS"
`,
				"sbx": `#!/bin/sh
[ "$1" = kit ] || exit 1
case "$2" in
  push)
    [ "$#" = 5 ] && [ "$3" = --sign ] &&
      [ "$4" = ./cloudsmith-repo ] && [ "$5" = "$EXPECTED_REF" ] || exit 1
    touch "$PUSHED"
    if [ "$SIGN_FAIL" = true ]; then
      echo "signing failed" >&2
      exit 1
    fi
    ;;
  verify)
    [ "$#" = 7 ] && [ "$3" = --certificate-identity ] &&
      [ "$4" = "https://github.com/$GITHUB_WORKFLOW_REF" ] &&
      [ "$5" = --certificate-oidc-issuer ] &&
      [ "$6" = https://token.actions.githubusercontent.com ] &&
      [ "$7" = "$EXPECTED_DIGEST_REF" ] || exit 1
    if [ "$VERIFY_FAIL" = true ]; then
      echo "verification failed" >&2
      exit 1
    fi
    touch "$VERIFIED"
    ;;
  *) exit 1 ;;
esac
`,
				"jq": `#!/bin/sh
cat >/dev/null
printf 'sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\n'
`,
			} {
				require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(body), 0o755))
			}
			ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, "bash", "-c", script)
			cmd.Dir = dir
			pushed := filepath.Join(dir, "pushed")
			verified := filepath.Join(dir, "verified")
			summary := filepath.Join(dir, "summary")
			digestRef := strings.TrimSuffix(ref, ":v0.1.0") + "@" + digest
			cmd.Env = append(os.Environ(),
				"PATH="+dir+":"+os.Getenv("PATH"),
				"CLOUDSMITH_NAMESPACE=example", "CLOUDSMITH_REPO=kits",
				"DOCKERHUB_PUBLISH_NAMESPACE=example", "EXPECTED_REF="+ref,
				"KIT_DIR=cloudsmith-repo", "KIT_VERSION=0.1.0",
				"GITHUB_STEP_SUMMARY="+summary, "GITHUB_WORKFLOW_REF="+workflowRef,
				"EXPECTED_DIGEST_REF="+digestRef, "VERIFIED="+verified,
				"SIGN_FAIL="+strconv.FormatBool(tc.signFail), "VERIFY_FAIL="+strconv.FormatBool(tc.verifyFail),
				"REPLACE_EXISTING="+strconv.FormatBool(tc.replace),
				"PUSHED="+pushed, "LOOKUP_ERROR="+tc.message, "LOOKUP_STATUS="+tc.status)
			out, err := cmd.CombinedOutput()
			if tc.push {
				require.FileExists(t, pushed)
				if tc.signFail || tc.verifyFail {
					require.Error(t, err, "%s", out)
					require.NoFileExists(t, verified)
					require.NoFileExists(t, summary)
					if tc.signFail {
						require.Contains(t, string(out), "signing failed")
					} else {
						require.Contains(t, string(out), "verification failed")
					}
				} else {
					require.NoError(t, err, "%s", out)
					require.FileExists(t, verified)
					report, err := os.ReadFile(summary)
					require.NoError(t, err)
					require.Equal(t, "Published and verified "+digestRef+"\n", string(report))
				}
			} else {
				require.Error(t, err, "%s", out)
				require.NoFileExists(t, pushed)
				if tc.status == "0" {
					require.Contains(t, string(out), "already exists")
				} else {
					require.Contains(t, string(out), tc.message)
				}
			}
		})
	}
}

func TestPublishingOIDCPermissions(t *testing.T) {
	for _, file := range []string{"publish.yml", "publish-dockerhub.yml"} {
		t.Run(file, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join("../.github/workflows", file))
			require.NoError(t, err)
			var workflow struct {
				Jobs map[string]struct {
					Permissions map[string]string
				}
			}
			require.NoError(t, yaml.Unmarshal(data, &workflow))
			require.Equal(t, "write", workflow.Jobs["publish"].Permissions["id-token"])
			require.Equal(t, "read", workflow.Jobs["publish"].Permissions["contents"])
		})
	}
}

func TestCloudsmithReplacementOptIn(t *testing.T) {
	data, err := os.ReadFile("../.github/workflows/publish.yml")
	require.NoError(t, err)
	var workflow struct {
		Jobs map[string]struct {
			Env map[string]string
		}
	}
	require.NoError(t, yaml.Unmarshal(data, &workflow))
	require.Equal(t, "${{ github.event_name == 'workflow_dispatch' && inputs.replace_existing || false }}",
		workflow.Jobs["publish"].Env["REPLACE_EXISTING"])
}

func TestDockerHubReleaseInputs(t *testing.T) {
	script := publishingStep(t, "publish-dockerhub.yml", "Check release inputs")
	for _, tc := range []struct {
		name      string
		tag       string
		namespace string
		token     string
		ok        bool
	}{
		{"repo", "cloudsmith-repo-v0.1.0", "cloudsmithbot", "test-token", true},
		{"cli", "cloudsmith-cli-v0.1.0", "cloudsmithbot", "test-token", true},
		{"firewall", "cloudsmith-dependency-firewall-v0.1.0", "example-org", "test-token", true},
		{"missing-token", "cloudsmith-repo-v0.1.0", "cloudsmithbot", "", false},
		{"bare-tag", "v0.1.0", "cloudsmithbot", "test-token", false},
		{"unknown-kit", "unknown-v0.1.0", "cloudsmithbot", "test-token", false},
		{"leading-zero", "cloudsmith-repo-v01.0.0", "cloudsmithbot", "test-token", false},
		{"namespace-path", "cloudsmith-repo-v0.1.0", "example/nested", "test-token", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, "bash", "-c", script)
			cmd.Env = append(os.Environ(), "TAG="+tc.tag, "DOCKERHUB_PUBLISH_USER=cloudsmithbot",
				"DOCKERHUB_PUBLISH_NAMESPACE="+tc.namespace, "DOCKERHUB_PUBLISH_TOKEN="+tc.token)
			out, err := cmd.CombinedOutput()
			if tc.ok {
				require.NoError(t, err, "%s", out)
			} else {
				require.Error(t, err, "%s", out)
				require.Contains(t, string(out), "::error::")
			}
		})
	}
}

func TestDockerHubMissingUser(t *testing.T) {
	script := publishingStep(t, "publish-dockerhub.yml", "Check release inputs")
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "bash", "-c", script)
	cmd.Env = append(os.Environ(), "TAG=cloudsmith-repo-v0.1.0",
		"DOCKERHUB_PUBLISH_NAMESPACE=cloudsmithbot", "DOCKERHUB_PUBLISH_TOKEN=test-token",
		"DOCKERHUB_PUBLISH_USER=")
	out, err := cmd.CombinedOutput()
	require.Error(t, err, "%s", out)
	require.Contains(t, string(out), "::error::Set the DOCKERHUB_PUBLISH_USER secret")
}

func TestDockerHubPublishingOptIn(t *testing.T) {
	data, err := os.ReadFile("../.github/workflows/publish-dockerhub.yml")
	require.NoError(t, err)
	var workflow struct {
		Jobs map[string]struct {
			If  string
			Env map[string]string
		}
	}
	require.NoError(t, yaml.Unmarshal(data, &workflow))
	job := workflow.Jobs["publish"]
	require.Equal(t, "github.event_name == 'workflow_dispatch' || vars.DOCKERHUB_PUBLISH_ENABLED == 'true'", job.If)
	require.Equal(t, "${{ secrets.DOCKERHUB_PUBLISH_USER }}", job.Env["DOCKERHUB_PUBLISH_USER"])
	require.Equal(t, "${{ vars.DOCKERHUB_PUBLISH_NAMESPACE || 'cloudsmithbot' }}", job.Env["DOCKERHUB_PUBLISH_NAMESPACE"])
}

func TestDockerHubAuthentication(t *testing.T) {
	script := publishingStep(t, "publish-dockerhub.yml", "Authenticate to Docker Hub")
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "docker"), []byte(`#!/bin/sh
[ "$*" = "login --username cloudsmithbot --password-stdin" ] || exit 1
[ "$(cat)" = test-token ] || exit 1
exit "$LOGIN_STATUS"
`), 0o755))
	for _, status := range []string{"0", "1"} {
		t.Run(status, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, "bash", "-c", script)
			cmd.Env = append(os.Environ(), "PATH="+dir+":"+os.Getenv("PATH"),
				"DOCKERHUB_PUBLISH_USER=cloudsmithbot",
				"DOCKERHUB_PUBLISH_TOKEN=test-token", "LOGIN_STATUS="+status)
			out, err := cmd.CombinedOutput()
			if status == "0" {
				require.NoError(t, err, "%s", out)
			} else {
				require.Error(t, err, "%s", out)
			}
			require.NotContains(t, string(out), "test-token")
		})
	}
}
