package tck_test

import (
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/docker/sbx-kits-contrib/tck"
	"github.com/stretchr/testify/require"
)

func TestKitContracts(t *testing.T) {
	for _, kit := range []string{"cloudsmith-repo", "cloudsmith-cli", "cloudsmith-dependency-firewall"} {
		t.Run(kit, func(t *testing.T) {
			suite, err := tck.NewSuiteFromDir(filepath.Join("..", kit))
			require.NoError(t, err)
			suite.RunValidationTests(t)
			suite.RunNetworkPolicyTests(t)
			suite.RunCredentialPolicyTests(t)
			suite.RunEnvironmentPolicyTests(t)
			suite.RunCommandsValidationTests(t)
			for domain := range suite.ExpectedServiceDomains {
				require.Contains(t, suite.ExpectedAllowedDomains, domain, "credential destination must be allowed")
			}
			for _, credential := range suite.Artifact.Credentials {
				require.NotNil(t, credential.ApiKey)
				require.True(t, credential.ApiKey.ProxyManaged)
			}
		})
	}
}

func TestRepositoryClientConfiguration(t *testing.T) {
	suite, err := tck.NewSuiteFromDir("../cloudsmith-repo")
	require.NoError(t, err)
	env := suite.Artifact.Environment.Variables
	require.Equal(t, "https://dl.cloudsmith.io/basic/cloudsmith/cli/python/simple/", env["PIP_INDEX_URL"])
	require.Equal(t, env["PIP_INDEX_URL"], env["UV_DEFAULT_INDEX"])
	require.Equal(t, "https://golang.cloudsmith.io/cloudsmith/cli/", env["GOPROXY"])
	require.Equal(t, "https://npm.cloudsmith.io/cloudsmith/cli/", env["YARN_NPM_REGISTRY_SERVER"])
	for _, variable := range []string{"HTTP_PROXY", "HTTPS_PROXY", "ALL_PROXY", "NO_PROXY", "JAVA_TOOL_OPTIONS"} {
		require.NotContains(t, env, variable)
		require.NotContains(t, env, strings.ToLower(variable))
	}
	files := map[string]string{}
	for _, file := range suite.Artifact.Files {
		reader, err := file.Open()
		require.NoError(t, err)
		content, err := io.ReadAll(reader)
		require.NoError(t, err)
		require.NoError(t, reader.Close())
		files[file.RelativePath] = string(content)
		require.NotContains(t, string(content), "${{ kit.args.")
	}
	require.Contains(t, files[".cargo/config.toml"], "replace-with")
	require.Contains(t, files[".cargo/config.toml"], "sparse+https://cargo.cloudsmith.io/cloudsmith/cli/")
	require.Contains(t, files[".m2/settings.xml"], "<mirrorOf>*</mirrorOf>")
	require.Contains(t, files[".nuget/NuGet/NuGet.Config"], "<clear")
	require.Contains(t, files[".yarnrc.yml"], "httpsProxy:")
	require.Contains(t, files[".yarnrc.yml"], "httpProxy:")
	require.Contains(t, files[".yarnrc.yml"], "${HTTP_PROXY}")
	require.Contains(t, files[".yarnrc.yml"], "${HTTPS_PROXY}")
}

func TestArgumentBoundaries(t *testing.T) {
	suite, err := tck.NewSuiteFromDir("../cloudsmith-repo")
	require.NoError(t, err)
	for name, arg := range suite.Artifact.Args {
		t.Run(name, func(t *testing.T) {
			pattern, err := regexp.Compile(arg.Pattern)
			require.NoError(t, err)
			require.NotNil(t, arg.Default)
			require.True(t, pattern.MatchString(*arg.Default))
			for _, unsafe := range []string{"'; touch injected; '", "$(id)", "host\nallow: '*'", "user@host", "host:443", "../escape", "host/path"} {
				require.False(t, pattern.MatchString(unsafe), "accepted %q", unsafe)
			}
			if name == "path" {
				for _, path := range []string{"", "/repo", "/org/repo"} {
					require.True(t, pattern.MatchString(path), "rejected %q", path)
				}
				require.False(t, pattern.MatchString("/org/repo/extra"))
			}
		})
	}
}

func TestRepositoryProbeFailures(t *testing.T) {
	suite, err := tck.NewSuiteFromDir("../cloudsmith-repo")
	require.NoError(t, err)
	require.NotEmpty(t, suite.Artifact.Commands.Install)
	for _, tc := range []struct {
		status string
		exit   string
		ok     bool
	}{
		{"200", "0", true},
		{"401", "0", false},
		{"403", "0", false},
		{"404", "0", false},
		{"302", "0", false},
		{"500", "0", false},
		{"000", "7", false},
	} {
		t.Run(tc.status, func(t *testing.T) {
			dir := t.TempDir()
			require.NoError(t, os.WriteFile(filepath.Join(dir, "curl"), []byte("#!/bin/sh\nprintf '%s' \"$TEST_STATUS\"\nexit \"$TEST_EXIT\"\n"), 0o755))
			ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, "sh", "-c", suite.Artifact.Commands.Install[0].Command)
			cmd.Env = append(os.Environ(), "PATH="+dir+":"+os.Getenv("PATH"), "TEST_STATUS="+tc.status, "TEST_EXIT="+tc.exit)
			out, err := cmd.CombinedOutput()
			if tc.ok {
				require.NoError(t, err, "%s", out)
			} else {
				require.Error(t, err, "%s", out)
				require.Contains(t, string(out), "[cloudsmith-repo]")
			}
		})
	}
}

func TestFirewallContract(t *testing.T) {
	suite, err := tck.NewSuiteFromDir("../cloudsmith-dependency-firewall")
	require.NoError(t, err)
	require.Empty(t, suite.ExpectedAllowedDomains)
	require.Empty(t, suite.Artifact.Credentials)
	for _, host := range []string{
		"pypi.org", "files.pythonhosted.org", "registry.npmjs.org", "registry.yarnpkg.com",
		"proxy.golang.org", "index.crates.io", "static.crates.io", "repo.maven.apache.org",
		"api.nuget.org", "globalcdn.nuget.org", "registry-1.docker.io", "auth.docker.io",
	} {
		require.Contains(t, suite.ExpectedDeniedDomains, host)
	}
	for _, host := range suite.ExpectedDeniedDomains {
		require.NotContains(t, host, "cloudsmith")
	}
}

func TestCLIContract(t *testing.T) {
	suite, err := tck.NewSuiteFromDir("../cloudsmith-cli")
	require.NoError(t, err)
	arg, ok := suite.Artifact.Args["cli-version"]
	require.True(t, ok)
	require.NotNil(t, arg.Default)
	require.Regexp(t, `^[0-9]+\.[0-9]+\.[0-9]+$`, *arg.Default, "default installs must be versioned")
	pattern, err := regexp.Compile(arg.Pattern)
	require.NoError(t, err)
	for _, value := range []string{"latest", "1.27.0", "2.0.0"} {
		require.True(t, pattern.MatchString(value))
	}
	for _, value := range []string{"v1.27.0", "'; id; '", "$(id)", "1.2.3\n"} {
		require.False(t, pattern.MatchString(value))
	}
	require.Equal(t, map[string]string{"api.cloudsmith.io": "cloudsmith-api-key"}, suite.ExpectedServiceDomains,
		"API credentials must not leak to installer downloads or upload storage")
	require.Len(t, suite.Artifact.Credentials, 1)
	key := suite.Artifact.Credentials[0].ApiKey
	require.NotNil(t, key)
	require.Len(t, key.Inject, 1)
	require.Equal(t, "api.cloudsmith.io", key.Inject[0].Domain)
	require.Equal(t, "X-Api-Key", key.Inject[0].Header)
	require.Equal(t, "%s", key.Inject[0].Format)
	require.Contains(t, suite.ExpectedAllowedDomains, "dl.cloudsmith.io")
	require.Contains(t, suite.ExpectedAllowedDomains, "cloudsmith-package-uploads-prd.s3-accelerate.amazonaws.com")
}
