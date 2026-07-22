package main

import (
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func listCloned(t *testing.T, fs afero.Fs) []SearchResult {
	t.Helper()

	config, err := NewConfig(fs)
	require.NoError(t, err)

	results, err := config.ListCloned()
	require.NoError(t, err)

	return results
}

func TestListClonedWithoutClonesReturnsEmptySlice(t *testing.T) {
	results := listCloned(t, writeConfig(t, baseConfig()))

	assert.NotNil(t, results)
	assert.Empty(t, results)
}

func TestListClonedReturnsOnlyClonedRepositories(t *testing.T) {
	fs := writeConfig(t, baseConfig())
	require.NoError(t, fs.MkdirAll("work/dynamic-routing/task-pool", 0o755))
	require.NoError(t, fs.MkdirAll("personal/static-routing/plan-assignment", 0o755))

	results := listCloned(t, fs)
	require.Len(t, results, 2)

	plan := results[0]
	assert.Equal(t, "example-user/plan-assignment", plan.Repository)
	assert.Equal(t, "plan-assignment", plan.Name)
	assert.Equal(t, []string{}, plan.Aliases)
	assert.Equal(t, "github", plan.Platform)
	assert.Equal(t, "personal", plan.Profile)
	assert.Equal(t, "personal/static-routing", plan.Directory)
	assert.Equal(t, "personal/static-routing/plan-assignment", plan.Path)

	taskPool := results[1]
	assert.Equal(t, "company/task-pool", taskPool.Repository)
	assert.Equal(t, []string{"TP"}, taskPool.Aliases)
	assert.Equal(t, "github", taskPool.Platform)
	assert.Equal(t, "work", taskPool.Profile)
	assert.Equal(t, "work/dynamic-routing/task-pool", taskPool.Path)
}

func TestListClonedIgnoresFileAtRepositoryPath(t *testing.T) {
	fs := writeConfig(t, baseConfig())
	require.NoError(t, afero.WriteFile(fs, "work/dynamic-routing/task-pool", []byte{}, 0o644))

	results := listCloned(t, fs)

	assert.Empty(t, results)
}
