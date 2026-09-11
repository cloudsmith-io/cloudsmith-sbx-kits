//go:build e2e

// End-to-end test against a real, installed sbx CLI. It creates a sandbox
// from the kit and asserts that the kit's declared environment, files and
// tmpfs mounts land inside it.
//
// The caller installs sbx and runs `sbx login` first. The `e2e` build tag
// keeps this out of the default `go test ./...` flow, so scripts/test-kit.sh
// stays runnable without sbx. Use scripts/test-kit-e2e.sh instead.

package tck_test

import (
	"os"
	"testing"

	"github.com/docker/sbx-kits-contrib/tck"
	"github.com/stretchr/testify/require"
)

// appName scopes every sbx call this repository's e2e run makes. The scoped
// daemon keeps its own sandboxes, policy, cache and credentials, so the run
// never touches the day-to-day sbx state. Keep it in sync with
// scripts/test-kit-e2e.sh.
const appName = "cloudsmith-sbx-kits-tck"

func TestE2EKit(t *testing.T) {
	kitPath := os.Getenv("KIT_UNDER_TEST")
	require.NotEmpty(t, kitPath, "KIT_UNDER_TEST must point at a kit directory")

	tck.RunE2EKit(t, kitPath, tck.E2EOptions{AppName: appName})
}
