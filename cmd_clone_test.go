package main

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func runCloneCommand(t *testing.T, fs afero.Fs, args ...string) error {
	t.Helper()

	return CloneCommand(fs).Run(context.Background(), append([]string{"clone"}, args...))
}

// Uses absolute SSH key paths so tests do not depend on the real home directory
func cloneTestFs(t *testing.T) afero.Fs {
	t.Helper()

	config := baseConfig()
	for name, profile := range config["profiles"].(map[string]any) {
		profile.(map[string]any)["ssh_key"] = "/keys/" + name
	}

	fs := writeConfig(t, config)
	for name := range config["profiles"].(map[string]any) {
		require.NoError(t, afero.WriteFile(fs, "/keys/"+name, []byte("key"), 0o600))
	}

	return fs
}

func stubGitClone(t *testing.T, stub func(args []string) error) *[][]string {
	t.Helper()

	original := gitClone
	t.Cleanup(func() { gitClone = original })

	calls := &[][]string{}
	gitClone = func(_ context.Context, args []string) error {
		*calls = append(*calls, args)

		if stub == nil {
			return nil
		}

		return stub(args)
	}

	return calls
}

func clonedPathOf(call []string) string {
	return call[len(call)-1]
}

func TestCloneCommandRequiresArguments(t *testing.T) {
	fs := writeConfig(t, baseConfig())

	err := runCloneCommand(t, fs)
	assert.ErrorContains(t, err, "expected at least one repository name or alias")
}

func TestCloneCommandAllRejectsArguments(t *testing.T) {
	fs := writeConfig(t, baseConfig())

	err := runCloneCommand(t, fs, "--all", "TP")
	assert.ErrorContains(t, err, "--all cannot be combined with repository arguments")
}

func TestCloneCommandDirRejectsArguments(t *testing.T) {
	fs := writeConfig(t, baseConfig())

	err := runCloneCommand(t, fs, "--dir", "work", "TP")
	assert.ErrorContains(t, err, "--dir cannot be combined with repository arguments")
}

func TestCloneCommandDirRejectsAll(t *testing.T) {
	fs := writeConfig(t, baseConfig())

	err := runCloneCommand(t, fs, "--all", "--dir", "work")
	assert.ErrorContains(t, err, "--all cannot be combined with --dir")
}

func TestCloneCommandUnknownRepository(t *testing.T) {
	fs := cloneTestFs(t)
	logs := captureLogs(t)
	calls := stubGitClone(t, nil)

	require.NoError(t, runCloneCommand(t, fs, "unknown"))
	assert.Contains(t, logs.String(), "no repository found with name or alias")
	assert.Empty(t, *calls)
}

func TestCloneCommandClonesRepository(t *testing.T) {
	fs := cloneTestFs(t)
	calls := stubGitClone(t, nil)

	require.NoError(t, runCloneCommand(t, fs, "TP"))
	require.Len(t, *calls, 1)
	assert.Equal(t, []string{
		"clone",
		"--config", "core.sshCommand=ssh -i '/keys/work' -o IdentitiesOnly=yes",
		"--config", "user.name=Jane Doe",
		"--config", "user.email=jane.doe@work-example.com",
		"git@github.com:company/task-pool.git", "work/dynamic-routing/task-pool",
	}, (*calls)[0])
}

func TestCloneCommandUsesRepositoryPlatformAndProfile(t *testing.T) {
	fs := cloneTestFs(t)
	calls := stubGitClone(t, nil)

	require.NoError(t, runCloneCommand(t, fs, "ATP", "plan-assignment"))
	require.Len(t, *calls, 2)
	assert.Contains(t, (*calls)[0], "git@git.company.com:company/active-task-pool.git")
	assert.Contains(t, (*calls)[1], "git@github.com:example-user/plan-assignment.git")
	assert.Contains(t, (*calls)[1], "core.sshCommand=ssh -i '/keys/personal' -o IdentitiesOnly=yes")
}

func TestCloneCommandAlreadyClonedRepository(t *testing.T) {
	fs := cloneTestFs(t)
	logs := captureLogs(t)
	calls := stubGitClone(t, nil)

	require.NoError(t, fs.MkdirAll("work/dynamic-routing/task-pool", 0o755))

	require.NoError(t, runCloneCommand(t, fs, "TP"))
	assert.Contains(t, logs.String(), "repository is already cloned")
	assert.Empty(t, *calls)
}

func TestCloneCommandDeduplicatesArguments(t *testing.T) {
	fs := cloneTestFs(t)
	calls := stubGitClone(t, nil)

	require.NoError(t, runCloneCommand(t, fs, "task-pool", "TP"))
	assert.Len(t, *calls, 1)
}

func TestCloneCommandMissingSSHKey(t *testing.T) {
	fs := cloneTestFs(t)
	logs := captureLogs(t)
	calls := stubGitClone(t, nil)

	require.NoError(t, fs.Remove("/keys/work"))

	err := runCloneCommand(t, fs, "TP")
	assert.ErrorContains(t, err, "failed to clone one or more repositories")
	assert.Contains(t, logs.String(), "SSH key does not exist")
	assert.Empty(t, *calls)
}

func TestCloneCommandContinuesAfterFailure(t *testing.T) {
	fs := cloneTestFs(t)
	calls := stubGitClone(t, func(args []string) error {
		if strings.Contains(clonedPathOf(args), "active-task-pool") {
			return errors.New("exit status 128")
		}

		return nil
	})

	err := runCloneCommand(t, fs, "ATP", "TP")
	assert.ErrorContains(t, err, "failed to clone one or more repositories")
	assert.Len(t, *calls, 2)
}

func TestCloneCommandAllClonesMissingRepositories(t *testing.T) {
	fs := cloneTestFs(t)
	calls := stubGitClone(t, nil)

	require.NoError(t, fs.MkdirAll("work/dynamic-routing/task-pool", 0o755))

	require.NoError(t, runCloneCommand(t, fs, "--all"))
	require.Len(t, *calls, 2)
	assert.Equal(t, "personal/static-routing/plan-assignment", clonedPathOf((*calls)[0]))
	assert.Equal(t, "work/dynamic-routing/active-task-pool", clonedPathOf((*calls)[1]))
}

func TestCloneCommandAllWithEverythingCloned(t *testing.T) {
	fs := cloneTestFs(t)
	logs := captureLogs(t)
	calls := stubGitClone(t, nil)

	for _, repoPath := range []string{
		"work/dynamic-routing/active-task-pool",
		"work/dynamic-routing/task-pool",
		"personal/static-routing/plan-assignment",
	} {
		require.NoError(t, fs.MkdirAll(repoPath, 0o755))
	}

	require.NoError(t, runCloneCommand(t, fs, "--all"))
	assert.Contains(t, logs.String(), "all repositories are already cloned")
	assert.Empty(t, *calls)
}

func TestCloneCommandDirClonesSubtree(t *testing.T) {
	fs := cloneTestFs(t)
	calls := stubGitClone(t, nil)

	require.NoError(t, runCloneCommand(t, fs, "--dir", "work"))
	require.Len(t, *calls, 2)
	assert.Equal(t, "work/dynamic-routing/active-task-pool", clonedPathOf((*calls)[0]))
	assert.Equal(t, "work/dynamic-routing/task-pool", clonedPathOf((*calls)[1]))
}

func TestCloneCommandDirUnknownDirectory(t *testing.T) {
	fs := cloneTestFs(t)
	logs := captureLogs(t)
	calls := stubGitClone(t, nil)

	require.NoError(t, runCloneCommand(t, fs, "--dir", "unknown"))
	assert.Contains(t, logs.String(), "no configured directory matches")
	assert.Empty(t, *calls)
}

func TestCloneCommandDirWithEverythingCloned(t *testing.T) {
	fs := cloneTestFs(t)
	logs := captureLogs(t)
	calls := stubGitClone(t, nil)

	require.NoError(t, fs.MkdirAll("work/dynamic-routing/active-task-pool", 0o755))
	require.NoError(t, fs.MkdirAll("work/dynamic-routing/task-pool", 0o755))

	require.NoError(t, runCloneCommand(t, fs, "--dir", "work"))
	assert.Contains(t, logs.String(), "all repositories are already cloned in directory")
	assert.Empty(t, *calls)
}
