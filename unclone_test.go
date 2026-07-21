package main

import (
	"context"
	"os/exec"
	"path"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func git(t *testing.T, dir string, args ...string) {
	t.Helper()

	command := exec.Command("git", args...)
	command.Dir = dir

	output, err := command.CombinedOutput()
	require.NoError(t, err, string(output))
}

func gitCommitAll(t *testing.T, dir, message string) {
	t.Helper()

	git(t, dir, "add", "-A")
	git(t, dir,
		"-c", "user.name=Test",
		"-c", "user.email=test@example.com",
		"-c", "commit.gpgsign=false",
		"commit", "--allow-empty", "-m", message,
	)
}

// Clones from a freshly created source repository, so the result has no
// uncommitted changes and no unpushed commits
func cloneCleanRepo(t *testing.T, repoPath string) {
	t.Helper()

	source := t.TempDir()
	git(t, source, "init", "-q")
	gitCommitAll(t, source, "initial commit")

	git(t, ".", "clone", "-q", source, repoPath)
}

func TestUnsavedWorkEmptyRepository(t *testing.T) {
	fs := afero.NewOsFs()
	repoPath := path.Join(t.TempDir(), "repo")
	git(t, ".", "init", "-q", repoPath)

	unsaved, err := unsavedWork(context.Background(), fs, repoPath)
	require.NoError(t, err)
	assert.Empty(t, unsaved)
}

func TestUnsavedWorkCleanRepository(t *testing.T) {
	fs := afero.NewOsFs()
	repoPath := path.Join(t.TempDir(), "repo")
	cloneCleanRepo(t, repoPath)

	unsaved, err := unsavedWork(context.Background(), fs, repoPath)
	require.NoError(t, err)
	assert.Empty(t, unsaved)
}

func TestUnsavedWorkUntrackedFile(t *testing.T) {
	fs := afero.NewOsFs()
	repoPath := path.Join(t.TempDir(), "repo")
	cloneCleanRepo(t, repoPath)

	require.NoError(t, afero.WriteFile(fs, path.Join(repoPath, "new.txt"), []byte("x"), 0o644))

	unsaved, err := unsavedWork(context.Background(), fs, repoPath)
	require.NoError(t, err)
	assert.Equal(t, "uncommitted changes", unsaved)
}

func TestUnsavedWorkUnpushedCommit(t *testing.T) {
	fs := afero.NewOsFs()
	repoPath := path.Join(t.TempDir(), "repo")
	cloneCleanRepo(t, repoPath)

	require.NoError(t, afero.WriteFile(fs, path.Join(repoPath, "new.txt"), []byte("x"), 0o644))
	gitCommitAll(t, repoPath, "unpushed commit")

	unsaved, err := unsavedWork(context.Background(), fs, repoPath)
	require.NoError(t, err)
	assert.Equal(t, "unpushed commits", unsaved)
}

func TestUnsavedWorkNonGitDirectory(t *testing.T) {
	fs := afero.NewOsFs()

	_, err := unsavedWork(context.Background(), fs, t.TempDir())
	assert.ErrorContains(t, err, "not a git repository")
}
