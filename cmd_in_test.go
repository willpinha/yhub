package main

import (
	"context"
	"encoding/json"
	"os"
	"path"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func runInCommand(t *testing.T, fs afero.Fs, args ...string) error {
	t.Helper()

	return InCommand(fs).Run(context.Background(), append([]string{"in"}, args...))
}

// The "in" command runs commands relative to the current directory, so tests
// that execute them need a real filesystem rooted at a temporary cwd
func chdirTempFs(t *testing.T) afero.Fs {
	t.Helper()

	orig, err := os.Getwd()
	require.NoError(t, err)

	require.NoError(t, os.Chdir(t.TempDir()))
	t.Cleanup(func() { require.NoError(t, os.Chdir(orig)) })

	fs := afero.NewOsFs()

	data, err := json.Marshal(baseConfig())
	require.NoError(t, err)
	require.NoError(t, afero.WriteFile(fs, configPath, data, 0o644))

	return fs
}

func TestInCommandRequiresRepositoryAndCommand(t *testing.T) {
	fs := writeConfig(t, baseConfig())

	for _, args := range [][]string{{}, {"task-pool"}} {
		err := runInCommand(t, fs, args...)
		assert.ErrorContains(t, err, "expected a repository name or alias followed by a command")
	}
}

func TestInCommandUnknownRepository(t *testing.T) {
	fs := writeConfig(t, baseConfig())

	err := runInCommand(t, fs, "unknown", "true")
	assert.ErrorContains(t, err, "no repository found with name or alias 'unknown'")
}

func TestInCommandNotClonedRepository(t *testing.T) {
	fs := writeConfig(t, baseConfig())

	err := runInCommand(t, fs, "TP", "true")
	assert.ErrorContains(t, err, "repository 'company/task-pool' is not cloned")
}

func TestInCommandRunsCommandInsideRepository(t *testing.T) {
	fs := chdirTempFs(t)

	repoPath := "work/dynamic-routing/task-pool"
	require.NoError(t, fs.MkdirAll(repoPath, 0o755))

	err := runInCommand(t, fs, "TP", "touch", "out.txt")
	require.NoError(t, err)

	exists, err := afero.Exists(fs, path.Join(repoPath, "out.txt"))
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestInCommandFindsRepositoryByName(t *testing.T) {
	fs := chdirTempFs(t)

	repoPath := "personal/static-routing/plan-assignment"
	require.NoError(t, fs.MkdirAll(repoPath, 0o755))

	err := runInCommand(t, fs, "plan-assignment", "touch", "out.txt")
	require.NoError(t, err)

	exists, err := afero.Exists(fs, path.Join(repoPath, "out.txt"))
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestInCommandPassesFlagsToCommand(t *testing.T) {
	fs := chdirTempFs(t)

	repoPath := "work/dynamic-routing/task-pool"
	require.NoError(t, fs.MkdirAll(repoPath, 0o755))

	err := runInCommand(t, fs, "TP", "sh", "-c", "echo hello > out.txt")
	require.NoError(t, err)

	content, err := afero.ReadFile(fs, path.Join(repoPath, "out.txt"))
	require.NoError(t, err)
	assert.Equal(t, "hello\n", string(content))
}

func TestInCommandFailedCommand(t *testing.T) {
	fs := chdirTempFs(t)

	repoPath := "work/dynamic-routing/task-pool"
	require.NoError(t, fs.MkdirAll(repoPath, 0o755))

	err := runInCommand(t, fs, "TP", "false")
	assert.ErrorContains(t, err, "command 'false' failed")
}
