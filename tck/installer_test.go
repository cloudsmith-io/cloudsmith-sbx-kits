package tck_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/docker/sbx-kits-contrib/tck"
	"github.com/stretchr/testify/require"
)

func TestCLIInstallerFailureCleanup(t *testing.T) {
	suite, err := tck.NewSuiteFromDir("../cloudsmith-cli")
	require.NoError(t, err)
	require.Len(t, suite.Artifact.Commands.Install, 1)

	for _, failure := range []string{"download", "checksum", "install", "missing-executable", "link", "version", "none"} {
		t.Run(failure, func(t *testing.T) {
			dir := t.TempDir()
			bin := filepath.Join(dir, "bin")
			stage := filepath.Join(dir, "stage")
			trace := filepath.Join(dir, "trace")
			require.NoError(t, os.Mkdir(bin, 0o755))
			stubs := map[string]string{
				"mktemp": `mkdir "$TEST_STAGE"
printf '%s\n' "$TEST_STAGE"
`,
				"curl": `printf 'download\n' >> "$TEST_TRACE"
[ "$TEST_FAILURE" != download ] || exit 7
while [ "$#" -gt 0 ]; do
  if [ "$1" = -o ]; then
    shift
    printf '#!/bin/sh\n' > "$1"
    exit 0
  fi
  shift
done
exit 2
`,
				"sha256sum": `cat >/dev/null
printf 'checksum\n' >> "$TEST_TRACE"
[ "$TEST_FAILURE" != checksum ]
`,
				"sh": `printf 'install\n' >> "$TEST_TRACE"
[ "$TEST_FAILURE" != install ] || exit 13
while [ "$#" -gt 0 ]; do
  if [ "$1" = --output-file ]; then
    shift
    if [ "$TEST_FAILURE" = missing-executable ]; then
      printf 'executable=%s/missing\n' "$TEST_BIN" > "$1"
    else
      printf 'executable=%s/cloudsmith\n' "$TEST_BIN" > "$1"
    fi
    exit 0
  fi
  shift
done
exit 2
`,
				"ln": `printf 'link\n' >> "$TEST_TRACE"
[ "$1" = -sfn ] && [ "$2" = "$TEST_BIN/cloudsmith" ] && [ "$3" = /usr/local/bin/cloudsmith ] || exit 2
[ "$TEST_FAILURE" != link ]
`,
				"cloudsmith": `printf 'version\n' >> "$TEST_TRACE"
[ "$1" = --version ] && [ "$TEST_FAILURE" != version ]
`,
			}
			for name, body := range stubs {
				require.NoError(t, os.WriteFile(filepath.Join(bin, name), []byte("#!/bin/sh\nset -eu\n"+body), 0o755))
			}
			ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, "/bin/sh", "-c", suite.Artifact.Commands.Install[0].Command)
			cmd.Env = append(os.Environ(), "PATH="+bin+":"+os.Getenv("PATH"),
				"TEST_FAILURE="+failure, "TEST_STAGE="+stage, "TEST_TRACE="+trace, "TEST_BIN="+bin)
			out, err := cmd.CombinedOutput()
			if failure == "none" {
				require.NoError(t, err, "%s", out)
			} else {
				require.Error(t, err, "%s", out)
			}
			_, err = os.Stat(stage)
			require.True(t, os.IsNotExist(err), "installer staging must be removed on success and failure")
			log, err := os.ReadFile(trace)
			require.NoError(t, err)
			if failure == "download" || failure == "checksum" {
				require.NotContains(t, string(log), "install\n", "unverified downloads must never execute")
			}
			if failure == "none" {
				require.Equal(t, "download\nchecksum\ninstall\nlink\nversion\n", string(log))
			}
		})
	}
}
