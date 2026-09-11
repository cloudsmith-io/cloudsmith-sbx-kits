package tck_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestE2EWrapperIsolation(t *testing.T) {
	root, err := filepath.Abs("..")
	require.NoError(t, err)
	for _, tc := range []struct {
		app string
		ok  bool
	}{
		{"cs-kits", true},
		{"cs-kits-review", true},
		{"default", false},
		{"cs-kits-toolongname", false},
	} {
		t.Run(tc.app, func(t *testing.T) {
			dir := t.TempDir()
			trace := filepath.Join(dir, "trace")
			require.NoError(t, os.WriteFile(filepath.Join(dir, "sbx"), []byte(`#!/bin/sh
printf 'sbx %s\n' "$*" >> "$TRACE"
[ "$1" = --app-name ] && [ "$2" = "$APP_NAME" ]
`), 0o755))
			require.NoError(t, os.WriteFile(filepath.Join(dir, "go"), []byte(`#!/bin/sh
printf 'go APP_NAME=%s KIT_UNDER_TEST=%s\n' "$APP_NAME" "$KIT_UNDER_TEST" >> "$TRACE"
[ -f "$SBX_E2E_NAME_LOG" ]
`), 0o755))
			ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, "bash", filepath.Join(root, "scripts/test-kit-e2e.sh"), "cloudsmith-repo")
			cmd.Dir = root
			cmd.Env = append(os.Environ(), "PATH="+dir+":"+os.Getenv("PATH"), "APP_NAME="+tc.app, "POLICY=deny-all", "TRACE="+trace)
			out, err := cmd.CombinedOutput()
			if !tc.ok {
				require.Error(t, err, "%s", out)
				_, err = os.Stat(trace)
				require.True(t, os.IsNotExist(err), "invalid app name must fail before invoking sbx")
				return
			}
			require.NoError(t, err, "%s", out)
			log, err := os.ReadFile(trace)
			require.NoError(t, err)
			require.Contains(t, string(log), "sbx --app-name "+tc.app+" policy init deny-all")
			require.Contains(t, string(log), "go APP_NAME="+tc.app+" KIT_UNDER_TEST="+filepath.Join(root, "cloudsmith-repo"))
		})
	}
}
