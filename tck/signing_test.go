package tck_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"
)

func TestAmendedSignedReleaseBundle(t *testing.T) {
	data, err := os.ReadFile("../.github/workflows/sign.yml")
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
	for _, step := range workflow.Jobs["sign"].Steps {
		if step.Name == "Commit signatures and prepare delivery" {
			script = step.Run
		}
	}
	require.NotEmpty(t, script)
	dir := t.TempDir()
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	run := func(name string, args ...string) []byte {
		t.Helper()
		cmd := exec.CommandContext(ctx, name, args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL="+os.DevNull, "GIT_CONFIG_NOSYSTEM=1")
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "%s", out)
		return out
	}
	run("git", "init", "--quiet")
	run("git", "config", "user.name", "Test Bot")
	run("git", "config", "user.email", "bot@example.com")
	run("git", "-c", "commit.gpgsign=false", "commit", "--allow-empty", "-m", "Initial release")
	original := string(run("git", "rev-parse", "HEAD"))
	keyPath := filepath.Join(dir, "key")
	run("ssh-keygen", "-q", "-t", "ed25519", "-N", "", "-f", keyPath)
	key, err := os.ReadFile(keyPath)
	require.NoError(t, err)
	for _, kit := range []string{"cloudsmith-repo", "cloudsmith-cli", "cloudsmith-dependency-firewall"} {
		require.NoError(t, os.Mkdir(filepath.Join(dir, kit), 0o700))
		require.NoError(t, os.WriteFile(filepath.Join(dir, kit, "kit.sig.bundle"), []byte("test bundle\n"), 0o600))
	}
	require.NoError(t, os.WriteFile(filepath.Join(dir, "sbx"), []byte(`#!/bin/sh
[ "$1" = kit ] && [ "$2" = inspect ] || exit 1
printf '{"version":"0.1.0"}\n'
`), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "gh"), []byte("#!/bin/sh\nexit 99\n"), 0o755))
	cmd := exec.CommandContext(ctx, "bash", "-c", script)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "PATH="+dir+":"+os.Getenv("PATH"),
		"GIT_CONFIG_GLOBAL="+os.DevNull, "GIT_CONFIG_NOSYSTEM=1",
		"GH_TOKEN=test-token", "BOT_NAME=Test Bot", "BOT_EMAIL=bot@example.com",
		"BOT_SIGNING_KEY="+string(key), "AMEND_RELEASE=true", "RUNNER_TEMP="+dir,
		"GITHUB_STEP_SUMMARY="+filepath.Join(dir, "summary"))
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "%s", out)
	require.NotEqual(t, original, string(run("git", "rev-parse", "HEAD")))
	require.Equal(t, "1\n", string(run("git", "rev-list", "--count", "HEAD")))
	require.FileExists(t, filepath.Join(dir, "signed-release.bundle"))
	run("git", "bundle", "verify", "signed-release.bundle")
	heads := string(run("git", "bundle", "list-heads", "signed-release.bundle"))
	for _, kit := range []string{"cloudsmith-repo", "cloudsmith-cli", "cloudsmith-dependency-firewall"} {
		require.Contains(t, heads, "refs/tags/"+kit+"-v0.1.0")
	}
	require.NotContains(t, string(out), "test-token")
	require.NotContains(t, string(out), string(key))
}
