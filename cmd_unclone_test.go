package main

import (
	"bytes"
	"context"
	"log/slog"
	"path"
	"strings"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func runUncloneCommand(t *testing.T, fs afero.Fs, args ...string) error {
	t.Helper()

	return UncloneCommand(fs).Run(context.Background(), append([]string{"unclone"}, args...))
}

func captureLogs(t *testing.T) *bytes.Buffer {
	t.Helper()

	var buffer bytes.Buffer

	original := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buffer, nil)))
	t.Cleanup(func() { slog.SetDefault(original) })

	return &buffer
}

func assertCloned(t *testing.T, fs afero.Fs, repoPath string, want bool) {
	t.Helper()

	cloned, err := afero.DirExists(fs, repoPath)
	require.NoError(t, err)
	assert.Equal(t, want, cloned)
}

func TestUncloneCommandRequiresArguments(t *testing.T) {
	fs := writeConfig(t, baseConfig())

	err := runUncloneCommand(t, fs)
	assert.ErrorContains(t, err, "expected at least one repository name or alias")
}

func TestUncloneCommandAllRejectsArguments(t *testing.T) {
	fs := writeConfig(t, baseConfig())

	err := runUncloneCommand(t, fs, "--all", "TP")
	assert.ErrorContains(t, err, "--all cannot be combined with repository arguments")
}

func TestUncloneCommandUnknownRepository(t *testing.T) {
	fs := writeConfig(t, baseConfig())
	logs := captureLogs(t)

	require.NoError(t, runUncloneCommand(t, fs, "unknown"))
	assert.Contains(t, logs.String(), "no repository found with name or alias")
}

func TestUncloneCommandNotClonedRepository(t *testing.T) {
	fs := writeConfig(t, baseConfig())
	logs := captureLogs(t)

	require.NoError(t, runUncloneCommand(t, fs, "TP"))
	assert.Contains(t, logs.String(), "repository is not cloned")
}

func TestUncloneCommandRemovesCleanRepository(t *testing.T) {
	fs := chdirTempFs(t)

	repoPath := "work/dynamic-routing/task-pool"
	cloneCleanRepo(t, repoPath)

	require.NoError(t, runUncloneCommand(t, fs, "TP"))
	assertCloned(t, fs, repoPath, false)
}

func TestUncloneCommandKeepsRepositoryWithUncommittedChanges(t *testing.T) {
	fs := chdirTempFs(t)
	logs := captureLogs(t)

	repoPath := "work/dynamic-routing/task-pool"
	cloneCleanRepo(t, repoPath)
	require.NoError(t, afero.WriteFile(fs, path.Join(repoPath, "new.txt"), []byte("x"), 0o644))

	require.NoError(t, runUncloneCommand(t, fs, "TP"))
	assertCloned(t, fs, repoPath, true)
	assert.Contains(t, logs.String(), "repository has unsaved work")
}

func TestUncloneCommandKeepsRepositoryWithUnpushedCommits(t *testing.T) {
	fs := chdirTempFs(t)

	repoPath := "work/dynamic-routing/task-pool"
	cloneCleanRepo(t, repoPath)
	require.NoError(t, afero.WriteFile(fs, path.Join(repoPath, "new.txt"), []byte("x"), 0o644))
	gitCommitAll(t, repoPath, "unpushed commit")

	require.NoError(t, runUncloneCommand(t, fs, "TP"))
	assertCloned(t, fs, repoPath, true)
}

func TestUncloneCommandForceRemovesRepositoryWithUnsavedWork(t *testing.T) {
	fs := chdirTempFs(t)

	repoPath := "work/dynamic-routing/task-pool"
	cloneCleanRepo(t, repoPath)
	require.NoError(t, afero.WriteFile(fs, path.Join(repoPath, "new.txt"), []byte("x"), 0o644))

	require.NoError(t, runUncloneCommand(t, fs, "--force", "TP"))
	assertCloned(t, fs, repoPath, false)
}

func TestUncloneCommandFailsOnNonGitDirectory(t *testing.T) {
	fs := chdirTempFs(t)
	logs := captureLogs(t)

	repoPath := "work/dynamic-routing/task-pool"
	require.NoError(t, fs.MkdirAll(repoPath, 0o755))

	err := runUncloneCommand(t, fs, "TP")
	assert.ErrorContains(t, err, "failed to unclone one or more repositories")
	assertCloned(t, fs, repoPath, true)
	assert.Contains(t, logs.String(), "not a git repository")
}

func TestUncloneCommandForceRemovesNonGitDirectory(t *testing.T) {
	fs := chdirTempFs(t)

	repoPath := "work/dynamic-routing/task-pool"
	require.NoError(t, fs.MkdirAll(repoPath, 0o755))

	require.NoError(t, runUncloneCommand(t, fs, "--force", "TP"))
	assertCloned(t, fs, repoPath, false)
}

func TestUncloneCommandDeduplicatesArguments(t *testing.T) {
	fs := chdirTempFs(t)
	logs := captureLogs(t)

	repoPath := "work/dynamic-routing/task-pool"
	cloneCleanRepo(t, repoPath)

	require.NoError(t, runUncloneCommand(t, fs, "task-pool", "TP"))
	assertCloned(t, fs, repoPath, false)
	assert.Equal(t, 1, strings.Count(logs.String(), "repository removed"))
	assert.NotContains(t, logs.String(), "repository is not cloned")
}

func TestUncloneCommandContinuesAfterFailure(t *testing.T) {
	fs := chdirTempFs(t)

	brokenPath := "work/dynamic-routing/active-task-pool"
	require.NoError(t, fs.MkdirAll(brokenPath, 0o755))

	repoPath := "work/dynamic-routing/task-pool"
	cloneCleanRepo(t, repoPath)

	err := runUncloneCommand(t, fs, "ATP", "TP")
	assert.ErrorContains(t, err, "failed to unclone one or more repositories")
	assertCloned(t, fs, brokenPath, true)
	assertCloned(t, fs, repoPath, false)
}

func TestUncloneCommandAllRemovesAllClonedRepositories(t *testing.T) {
	fs := chdirTempFs(t)
	logs := captureLogs(t)

	taskPoolPath := "work/dynamic-routing/task-pool"
	planPath := "personal/static-routing/plan-assignment"
	cloneCleanRepo(t, taskPoolPath)
	cloneCleanRepo(t, planPath)

	require.NoError(t, runUncloneCommand(t, fs, "--all"))
	assertCloned(t, fs, taskPoolPath, false)
	assertCloned(t, fs, planPath, false)
	assert.Equal(t, 2, strings.Count(logs.String(), "repository removed"))
}

func TestUncloneCommandAllKeepsRepositoriesWithUnsavedWork(t *testing.T) {
	fs := chdirTempFs(t)

	taskPoolPath := "work/dynamic-routing/task-pool"
	planPath := "personal/static-routing/plan-assignment"
	cloneCleanRepo(t, taskPoolPath)
	cloneCleanRepo(t, planPath)
	require.NoError(t, afero.WriteFile(fs, path.Join(planPath, "new.txt"), []byte("x"), 0o644))

	require.NoError(t, runUncloneCommand(t, fs, "--all"))
	assertCloned(t, fs, taskPoolPath, false)
	assertCloned(t, fs, planPath, true)
}

func TestUncloneCommandAllWithNothingCloned(t *testing.T) {
	fs := writeConfig(t, baseConfig())
	logs := captureLogs(t)

	require.NoError(t, runUncloneCommand(t, fs, "--all"))
	assert.Contains(t, logs.String(), "no repositories are cloned")
}
