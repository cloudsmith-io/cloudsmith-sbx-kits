package tck_test

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/docker/sbx-kits-contrib/tck"
	"github.com/stretchr/testify/require"
)

func kitWithArgs(t *testing.T, kit string, args map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.CopyFS(dir, os.DirFS(filepath.Join("..", kit))))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "testdata"), 0o755))
	data, err := json.Marshal(map[string]any{"args": args})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "testdata", "tck.yaml"), data, 0o600))
	return dir
}

func TestRepositoryDomainRendering(t *testing.T) {
	for _, tc := range []struct {
		name string
		path string
		host string
	}{
		{"standard", "/org/repo", ""},
		{"namespace-domain", "/repo", "packages.example.com"},
		{"repository-domain", "", "repository.example.com"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			args := map[string]string{"path": tc.path}
			hosts := map[string]string{
				"dl-host": "dl.cloudsmith.io", "redirect-host": "dl.cloudsmith.io",
				"npm-host": "npm.cloudsmith.io", "go-host": "golang.cloudsmith.io",
				"cargo-host": "cargo.cloudsmith.io", "nuget-host": "nuget.cloudsmith.io",
				"docker-host": "docker.cloudsmith.io",
			}
			if tc.host != "" {
				for name := range hosts {
					hosts[name] = tc.host
					args[name] = tc.host
				}
				// Redirect resources may retain the standard authenticated host.
				hosts["redirect-host"] = "dl.cloudsmith.io"
				args["redirect-host"] = hosts["redirect-host"]
			}
			suite, err := tck.NewSuiteFromDir(kitWithArgs(t, "cloudsmith-repo", args))
			require.NoError(t, err)
			env := suite.Artifact.Environment.Variables
			index := "https://" + hosts["dl-host"] + "/basic" + tc.path + "/python/simple/"
			require.Equal(t, index, env["PIP_INDEX_URL"])
			require.Equal(t, index, env["UV_DEFAULT_INDEX"])
			registry := "https://" + hosts["npm-host"] + tc.path + "/"
			require.Equal(t, registry, env["NPM_CONFIG_REGISTRY"])
			require.Equal(t, registry, env["YARN_NPM_REGISTRY_SERVER"])
			require.Equal(t, "https://"+hosts["go-host"]+tc.path+"/", env["GOPROXY"])
			require.Contains(t, suite.Artifact.Commands.Install[0].Command, index)
			require.Contains(t, suite.Artifact.Commands.Install[1].Command, registry)
			require.Contains(t, suite.Artifact.Commands.Install[1].Command, "--location=user")
			for _, host := range hosts {
				require.Contains(t, suite.ExpectedAllowedDomains, host)
				require.Contains(t, suite.ExpectedServiceDomains, host)
			}
			expectedFiles := map[string]string{
				".cargo/config.toml":        "sparse+https://" + hosts["cargo-host"] + tc.path + "/",
				".m2/settings.xml":          "https://" + hosts["dl-host"] + "/basic" + tc.path + "/maven/",
				".nuget/NuGet/NuGet.Config": "https://" + hosts["nuget-host"] + tc.path + "/v3/index.json",
			}
			for _, file := range suite.Artifact.Files {
				reader, err := file.Open()
				require.NoError(t, err)
				content, err := io.ReadAll(reader)
				require.NoError(t, err)
				require.NoError(t, reader.Close())
				require.NotContains(t, string(content), "${{ kit.args.")
				if file.RelativePath == ".yarnrc.yml" {
					require.Contains(t, string(content), "${HTTP_PROXY}")
					require.Contains(t, string(content), "${HTTPS_PROXY}")
				}
				if expected, ok := expectedFiles[file.RelativePath]; ok {
					require.Contains(t, string(content), expected)
					delete(expectedFiles, file.RelativePath)
				}
			}
			require.Empty(t, expectedFiles, "expected client files are missing")
		})
	}
}

func TestTCKRejectsUnsafeArguments(t *testing.T) {
	for _, tc := range []struct {
		kit string
		arg string
		bad string
	}{
		{"cloudsmith-repo", "path", "/org/repo'; id; '"},
		{"cloudsmith-repo", "path", "/org/$(id)"},
		{"cloudsmith-repo", "dl-host", "example.com\npermissions: {}"},
		{"cloudsmith-repo", "npm-host", "user@example.com"},
		{"cloudsmith-repo", "redirect-host", "example.com'; id; '"},
		{"cloudsmith-repo", "redirect-host", "https://example.com/path"},
		{"cloudsmith-repo", "redirect-host", "example.com\nallow: '*'"},
		{"cloudsmith-cli", "cli-version", "1.27.0; id"},
		{"cloudsmith-cli", "cli-version", "$(id)"},
	} {
		t.Run(tc.kit+"/"+tc.arg+"/"+tc.bad, func(t *testing.T) {
			_, err := tck.NewSuiteFromDir(kitWithArgs(t, tc.kit, map[string]string{tc.arg: tc.bad}))
			require.Error(t, err)
		})
	}
}
