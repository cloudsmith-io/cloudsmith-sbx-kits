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
	"regexp"
	"testing"

	"github.com/docker/sbx-kits-contrib/tck"
	"github.com/stretchr/testify/require"
)

func TestE2EKit(t *testing.T) {
	kitPath := os.Getenv("KIT_UNDER_TEST")
	require.NotEmpty(t, kitPath, "KIT_UNDER_TEST must point at a kit directory")

	appName := os.Getenv("APP_NAME")
	if appName == "" {
		appName = "cs-kits"
	}
	require.Regexp(t, regexp.MustCompile(`^cs-kits(-[a-z0-9]{1,8})?$`), appName,
		"use a dedicated cs-kits test daemon")
	tck.RunE2EKit(t, kitPath, tck.E2EOptions{AppName: appName})
}
