package tck_test

import (
	"regexp"
	"testing"

	"github.com/docker/sbx-kits-contrib/tck"
	"github.com/stretchr/testify/require"
)

func TestRedirectHostContract(t *testing.T) {
	defaults, err := tck.NewSuiteFromDir("../cloudsmith-repo")
	require.NoError(t, err)
	arg, ok := defaults.Artifact.Args["redirect-host"]
	require.True(t, ok, "redirected NuGet resources need an independently authenticated host")
	require.NotNil(t, arg.Default)
	require.Equal(t, "dl.cloudsmith.io", *arg.Default)
	pattern, err := regexp.Compile(arg.Pattern)
	require.NoError(t, err)
	for _, host := range []string{"dl.cloudsmith.io", "dl-prod.example.cloudsmith.sh", "downloads.example.com"} {
		require.True(t, pattern.MatchString(host), "rejected %q", host)
	}
	for _, host := range []string{"", "*", "https://example.com", "example.com/path", "user@example.com", "example.com:443", "$(id)", "example.com\n"} {
		require.False(t, pattern.MatchString(host), "accepted %q", host)
	}

	const redirectHost = "dl-prod.example.cloudsmith.sh"
	suite, err := tck.NewSuiteFromDir(kitWithArgs(t, "cloudsmith-repo", map[string]string{
		"path": "/org/repo", "redirect-host": redirectHost,
	}))
	require.NoError(t, err)
	require.Contains(t, suite.ExpectedAllowedDomains, redirectHost)
	require.Equal(t, "cloudsmith-entitlement-token", suite.ExpectedServiceDomains[redirectHost],
		"NuGet resources on the redirect host require entitlement authentication")
	matches := 0
	for _, credential := range suite.Artifact.Credentials {
		require.NotNil(t, credential.ApiKey)
		for _, injection := range credential.ApiKey.Inject {
			if injection.Domain == redirectHost {
				matches++
				require.Equal(t, "Authorization", injection.Header)
				require.Equal(t, "Bearer %s", injection.Format)
			}
		}
	}
	require.Equal(t, 1, matches)
	require.Contains(t, suite.ExpectedServiceDomains, "dl.cloudsmith.io")
	require.Contains(t, suite.ExpectedServiceDomains, "npm.cloudsmith.io")
	env := suite.Artifact.Environment.Variables
	require.Equal(t, "https://npm.cloudsmith.io/org/repo/", env["NPM_CONFIG_REGISTRY"])
	require.Equal(t, "https://dl.cloudsmith.io/basic/org/repo/python/simple/", env["PIP_INDEX_URL"])
	require.Equal(t, "https://golang.cloudsmith.io/org/repo/", env["GOPROXY"])
	for _, value := range env {
		require.NotContains(t, value, redirectHost, "the redirect host must not replace package-client endpoints")
	}
}
