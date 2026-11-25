package tck_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"
)

// A change to any kit can make sign.yml close its signature pull request, so that change must start a new release run.
func TestReleaseRunsWhenAnySignedKitInputChanges(t *testing.T) {
	data, err := os.ReadFile("../.github/workflows/release.yml")
	require.NoError(t, err)
	var workflow struct {
		On struct {
			Push struct {
				Paths []string
			}
		} `yaml:"on"`
	}
	require.NoError(t, yaml.Unmarshal(data, &workflow))
	require.Subset(t, workflow.On.Push.Paths, []string{"*/spec.yaml", "*/files/**"})
}
