package main

import (
	"context"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCloneURL(t *testing.T) {
	assert.Equal(t, "git@github.com:owner/repo.git", cloneURL("github.com", "owner/repo"))
}

func TestCloneURLNestedGroups(t *testing.T) {
	assert.Equal(t, "git@gitlab.com:group/subgroup/repo.git", cloneURL("gitlab.com", "group/subgroup/repo"))
}

func TestSSHCommandQuotesKeyPath(t *testing.T) {
	assert.Equal(t, "ssh -i '/keys/my key' -o IdentitiesOnly=yes", sshCommand("/keys/my key"))
}

func TestSSHCommandEscapesSingleQuotes(t *testing.T) {
	assert.Equal(t, `ssh -i '/keys/jane'\''s key' -o IdentitiesOnly=yes`, sshCommand("/keys/jane's key"))
}

func TestExpandHomePathTilde(t *testing.T) {
	home, err := os.UserHomeDir()
	require.NoError(t, err)

	expanded, err := expandHomePath("~/.ssh/id_rsa")
	require.NoError(t, err)
	assert.Equal(t, path.Join(home, ".ssh/id_rsa"), expanded)
}

func TestExpandHomePathLiteral(t *testing.T) {
	for _, p := range []string{"/keys/id_rsa", "keys/id_rsa", "~user/id_rsa"} {
		expanded, err := expandHomePath(p)
		require.NoError(t, err)
		assert.Equal(t, p, expanded)
	}
}

func TestCloneArgs(t *testing.T) {
	profile := Profile{UserName: "Jane Doe", UserEmail: "jane.doe@example.com"}

	assert.Equal(t, []string{
		"clone",
		"--config", "core.sshCommand=ssh -i '/keys/id_rsa' -o IdentitiesOnly=yes",
		"--config", "user.name=Jane Doe",
		"--config", "user.email=jane.doe@example.com",
		"git@github.com:owner/repo.git", "dir/repo",
	}, cloneArgs("git@github.com:owner/repo.git", "dir/repo", profile, "/keys/id_rsa"))
}

// Clones from a local source repository, so the persisted --config values can
// be verified without network access
func TestGitClonePersistsConfig(t *testing.T) {
	source := t.TempDir()
	git(t, source, "init", "-q")
	gitCommitAll(t, source, "initial commit")

	ctx := context.Background()
	repoPath := path.Join(t.TempDir(), "repo")
	profile := Profile{UserName: "Jane Doe", UserEmail: "jane.doe@example.com"}

	require.NoError(t, gitClone(ctx, cloneArgs(source, repoPath, profile, "/keys/id_rsa")))

	for key, want := range map[string]string{
		"core.sshCommand": "ssh -i '/keys/id_rsa' -o IdentitiesOnly=yes",
		"user.name":       "Jane Doe",
		"user.email":      "jane.doe@example.com",
	} {
		value, err := gitOutput(ctx, repoPath, "config", "--local", key)
		require.NoError(t, err)
		assert.Equal(t, want, value)
	}
}
