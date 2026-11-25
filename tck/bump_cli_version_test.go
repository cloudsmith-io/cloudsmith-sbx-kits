// Copyright 2026 Cloudsmith Ltd

package tck_test

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestBumpCLIVersion(t *testing.T) {
	const spec = "version: \"0.1.0\"\nargs:\n  cli-version:\n    default: \"1.27.0\"\n"
	const changelog = "# Changelog\n\n## [0.1.0] - 2026-09-11\n\n### Added\n\n- The first release.\n"
	bumpedSpec := "version: \"0.1.1\"\nargs:\n  cli-version:\n    default: \"1.28.0\"\n"
	bumpedChangelog := "# Changelog\n\n## [0.1.1] - " + time.Now().UTC().Format(time.DateOnly) +
		"\n\n### Changed\n\n- Pin the default `cli-version` to Cloudsmith CLI 1.28.0.\n\n" +
		"## [0.1.0] - 2026-09-11\n\n### Added\n\n- The first release.\n"
	script, err := filepath.Abs("../scripts/bump-cli-version.sh")
	require.NoError(t, err)
	for _, tc := range []struct {
		name      string
		version   string
		stdout    string
		spec      string
		changelog string
		ok        bool
	}{
		{"newer", "1.28.0", "0.1.1\n", bumpedSpec, bumpedChangelog, true},
		{"older", "1.26.9", "", spec, changelog, true},
		{"new-major", "2.0.0", "", spec, changelog, true},
		{"not-a-release", "latest", "", spec, changelog, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			kit := filepath.Join(dir, "cloudsmith-cli")
			require.NoError(t, os.Mkdir(kit, 0o755))
			require.NoError(t, os.WriteFile(filepath.Join(kit, "spec.yaml"), []byte(spec), 0o644))
			require.NoError(t, os.WriteFile(filepath.Join(kit, "CHANGELOG.md"), []byte(changelog), 0o644))
			ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
			defer cancel()
			var stdout, stderr bytes.Buffer
			cmd := exec.CommandContext(ctx, "bash", script, tc.version)
			cmd.Dir = dir
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr
			err := cmd.Run()
			if tc.ok {
				require.NoError(t, err, "%s", stderr.String())
			} else {
				require.Error(t, err)
			}
			require.Equal(t, tc.stdout, stdout.String())
			if tc.stdout == "" {
				require.NotEmpty(t, stderr.String())
			}
			gotSpec, err := os.ReadFile(filepath.Join(kit, "spec.yaml"))
			require.NoError(t, err)
			require.Equal(t, tc.spec, string(gotSpec))
			gotChangelog, err := os.ReadFile(filepath.Join(kit, "CHANGELOG.md"))
			require.NoError(t, err)
			require.Equal(t, tc.changelog, string(gotChangelog))
		})
	}
}
