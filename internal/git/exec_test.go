package git_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/willpinha/yhub/internal/git"
)

func gitExec(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=Test",
		"GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=Test",
		"GIT_COMMITTER_EMAIL=test@example.com",
	)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "git %v: %s", args, out)
}

func makeSourceRepo(t *testing.T) string {
	t.Helper()
	src := t.TempDir()
	gitExec(t, src, "init", "-b", "main")
	gitExec(t, src, "config", "user.name", "Test")
	gitExec(t, src, "config", "user.email", "test@example.com")
	require.NoError(t, os.WriteFile(filepath.Join(src, "README"), []byte("hello"), 0o644))
	gitExec(t, src, "add", ".")
	gitExec(t, src, "commit", "-m", "init")
	return src
}

func TestExecGit_Integration(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}

	ctx := context.Background()
	g := git.New()

	src := makeSourceRepo(t)
	dest := filepath.Join(t.TempDir(), "clone")

	t.Run("Clone", func(t *testing.T) {
		err := g.Clone(ctx, src, dest, nil)
		require.NoError(t, err)

		_, err = os.Stat(filepath.Join(dest, ".git"))
		require.NoError(t, err, ".git directory should exist after clone")
	})

	t.Run("SetConfig", func(t *testing.T) {
		err := g.SetConfig(ctx, dest, "user.name", "CloneUser")
		require.NoError(t, err)

		out, err := exec.Command("git", "-C", dest, "config", "--local", "user.name").Output()
		require.NoError(t, err)
		assert.Equal(t, "CloneUser\n", string(out))
	})

	t.Run("HasUncommittedChanges_CleanRepo", func(t *testing.T) {
		dirty, err := g.HasUncommittedChanges(ctx, dest)
		require.NoError(t, err)
		assert.False(t, dirty)
	})

	t.Run("HasUncommittedChanges_DirtyRepo", func(t *testing.T) {
		require.NoError(t, os.WriteFile(filepath.Join(dest, "new.txt"), []byte("change"), 0o644))
		dirty, err := g.HasUncommittedChanges(ctx, dest)
		require.NoError(t, err)
		assert.True(t, dirty)

		gitExec(t, dest, "checkout", "--", ".")
		require.NoError(t, os.Remove(filepath.Join(dest, "new.txt")))
	})

	t.Run("HasUnpushedCommits_FreshClone", func(t *testing.T) {
		unpushed, err := g.HasUnpushedCommits(ctx, dest)
		require.NoError(t, err)
		assert.False(t, unpushed, "fresh clone should have no unpushed commits")
	})

	t.Run("HasUnpushedCommits_AfterLocalCommit", func(t *testing.T) {
		require.NoError(t, os.WriteFile(filepath.Join(dest, "local.txt"), []byte("local"), 0o644))
		gitExec(t, dest, "add", ".")
		gitExec(t, dest, "commit", "-m", "local commit")

		unpushed, err := g.HasUnpushedCommits(ctx, dest)
		require.NoError(t, err)
		assert.True(t, unpushed, "local commit not pushed should be reported")
	})
}
