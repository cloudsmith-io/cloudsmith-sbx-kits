// Run Docker's kit TCK against the kit named by the KIT environment
// variable. The suite is consumed from the published
// github.com/docker/sbx-kits-contrib module, so this repository tests its
// kits against the same harness the upstream kit repository uses instead of
// keeping a copy that drifts.

package tck_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/docker/sbx-kits-contrib/tck"
	"github.com/stretchr/testify/require"
)

// TestKitTCK loads the kit at $KIT and runs the full TCK suite against it:
// spec validation, the network, credential and environment policies, the
// setup commands, and container assertions through testcontainers.
//
// KIT must be an absolute path. `go test ./tck/...` sets the working
// directory to ./tck/, so a relative KIT resolves against ./tck/. Use
// scripts/test-kit.sh, which anchors the path for you.
func TestKitTCK(t *testing.T) {
	kitPath := os.Getenv("KIT")
	if kitPath == "" {
		t.Skip("KIT not set — use scripts/test-kit.sh or the per-kit CI matrix")
	}

	absKit, err := filepath.Abs(kitPath)
	require.NoError(t, err, "resolve KIT=%q", kitPath)

	info, err := os.Stat(absKit)
	require.NoErrorf(t, err, "stat KIT=%q", absKit)
	require.Truef(t, info.IsDir(), "KIT=%q must be a directory", absKit)

	t.Run(filepath.Base(absKit), func(t *testing.T) {
		suite, err := tck.NewSuiteFromDir(absKit)
		require.NoErrorf(t, err, "derive suite for %q", absKit)
		suite.RunAll(t)
	})
}
