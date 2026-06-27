package cli

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/willpinha/yhub/internal/config"
	"github.com/willpinha/yhub/internal/git/gittest"
)

func makeUncloneCfg() *config.Config {
	return &config.Config{
		RepositoriesDir: "repos",
		Groups: map[string][]config.Repository{
			"hello": {{Alias: "WR", Name: "world", Repository: "org/world"}},
		},
	}
}

func TestRunUnclone_CleanRepo_Removed(t *testing.T) {
	fs := afero.NewMemMapFs()
	require.NoError(t, fs.MkdirAll("repos/hello/world", 0o755))
	g := &gittest.FakeGit{}
	cfg := makeUncloneCfg()

	selected := []selectedRepo{{Group: "hello", Repo: cfg.Groups["hello"][0]}}
	var buf bytes.Buffer

	result := runUnclone(context.Background(), &buf, fs, g, cfg, selected, false)

	assert.Equal(t, 1, result.Removed)
	assert.Equal(t, 0, result.Skipped)
	assert.Equal(t, 0, result.Failed)

	exists, err := afero.DirExists(fs, "repos/hello/world")
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestRunUnclone_NotCloned_Skipped(t *testing.T) {
	fs := afero.NewMemMapFs()
	g := &gittest.FakeGit{}
	cfg := makeUncloneCfg()

	selected := []selectedRepo{{Group: "hello", Repo: cfg.Groups["hello"][0]}}
	var buf bytes.Buffer

	result := runUnclone(context.Background(), &buf, fs, g, cfg, selected, false)

	assert.Equal(t, 0, result.Removed)
	assert.Equal(t, 1, result.Skipped)
	assert.Equal(t, 0, result.Failed)
	assert.Contains(t, buf.String(), "not cloned, skipping")
}

func TestRunUnclone_UncommittedChanges_Skipped(t *testing.T) {
	fs := afero.NewMemMapFs()
	require.NoError(t, fs.MkdirAll("repos/hello/world", 0o755))
	g := &gittest.FakeGit{
		HasUncommittedChangesFunc: func(_ context.Context, _ string) (bool, error) {
			return true, nil
		},
	}
	cfg := makeUncloneCfg()

	selected := []selectedRepo{{Group: "hello", Repo: cfg.Groups["hello"][0]}}
	var buf bytes.Buffer

	result := runUnclone(context.Background(), &buf, fs, g, cfg, selected, false)

	assert.Equal(t, 0, result.Removed)
	assert.Equal(t, 1, result.Skipped)
	assert.Equal(t, 0, result.Failed)

	exists, err := afero.DirExists(fs, "repos/hello/world")
	require.NoError(t, err)
	assert.True(t, exists)
	assert.Contains(t, buf.String(), "has uncommitted changes")
}

func TestRunUnclone_UnpushedCommits_Skipped(t *testing.T) {
	fs := afero.NewMemMapFs()
	require.NoError(t, fs.MkdirAll("repos/hello/world", 0o755))
	g := &gittest.FakeGit{
		HasUncommittedChangesFunc: func(_ context.Context, _ string) (bool, error) {
			return false, nil
		},
		HasUnpushedCommitsFunc: func(_ context.Context, _ string) (bool, error) {
			return true, nil
		},
	}
	cfg := makeUncloneCfg()

	selected := []selectedRepo{{Group: "hello", Repo: cfg.Groups["hello"][0]}}
	var buf bytes.Buffer

	result := runUnclone(context.Background(), &buf, fs, g, cfg, selected, false)

	assert.Equal(t, 0, result.Removed)
	assert.Equal(t, 1, result.Skipped)
	assert.Equal(t, 0, result.Failed)

	exists, err := afero.DirExists(fs, "repos/hello/world")
	require.NoError(t, err)
	assert.True(t, exists)
	assert.Contains(t, buf.String(), "has unpushed commits")
}

func TestRunUnclone_Force_BypassesGitChecks(t *testing.T) {
	fs := afero.NewMemMapFs()
	require.NoError(t, fs.MkdirAll("repos/hello/world", 0o755))
	g := &gittest.FakeGit{
		HasUncommittedChangesFunc: func(_ context.Context, _ string) (bool, error) {
			return true, nil
		},
	}
	cfg := makeUncloneCfg()

	selected := []selectedRepo{{Group: "hello", Repo: cfg.Groups["hello"][0]}}
	var buf bytes.Buffer

	result := runUnclone(context.Background(), &buf, fs, g, cfg, selected, true)

	assert.Equal(t, 1, result.Removed)
	assert.Equal(t, 0, result.Skipped)
	assert.Equal(t, 0, result.Failed)
	assert.Empty(t, g.HasUncommittedChangesRepo)
	assert.Empty(t, g.HasUnpushedCommitsRepo)

	exists, err := afero.DirExists(fs, "repos/hello/world")
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestRunUnclone_StatusCheckError_ConservativeSkip(t *testing.T) {
	fs := afero.NewMemMapFs()
	require.NoError(t, fs.MkdirAll("repos/hello/world", 0o755))
	g := &gittest.FakeGit{
		HasUncommittedChangesFunc: func(_ context.Context, _ string) (bool, error) {
			return false, errors.New("git error")
		},
	}
	cfg := makeUncloneCfg()

	selected := []selectedRepo{{Group: "hello", Repo: cfg.Groups["hello"][0]}}
	var buf bytes.Buffer

	result := runUnclone(context.Background(), &buf, fs, g, cfg, selected, false)

	assert.Equal(t, 0, result.Removed)
	assert.Equal(t, 1, result.Skipped)
	assert.Equal(t, 0, result.Failed)

	exists, err := afero.DirExists(fs, "repos/hello/world")
	require.NoError(t, err)
	assert.True(t, exists)
	assert.Contains(t, buf.String(), "cannot check status")
	assert.Contains(t, buf.String(), "use --force to remove")
}

func TestRunUnclone_MultiRepo_MixedStates(t *testing.T) {
	fs := afero.NewMemMapFs()
	require.NoError(t, fs.MkdirAll("repos/work/repo1", 0o755))
	require.NoError(t, fs.MkdirAll("repos/work/repo2", 0o755))

	dirtyRepos := map[string]bool{"repos/work/repo1": true}
	g := &gittest.FakeGit{
		HasUncommittedChangesFunc: func(_ context.Context, repoPath string) (bool, error) {
			return dirtyRepos[repoPath], nil
		},
	}
	cfg := &config.Config{
		RepositoriesDir: "repos",
		Groups: map[string][]config.Repository{
			"work": {
				{Alias: "R1", Name: "repo1", Repository: "org/repo1"},
				{Alias: "R2", Name: "repo2", Repository: "org/repo2"},
				{Alias: "R3", Name: "repo3", Repository: "org/repo3"},
			},
		},
	}

	selected := []selectedRepo{
		{Group: "work", Repo: cfg.Groups["work"][0]},
		{Group: "work", Repo: cfg.Groups["work"][1]},
		{Group: "work", Repo: cfg.Groups["work"][2]},
	}
	var buf bytes.Buffer

	result := runUnclone(context.Background(), &buf, fs, g, cfg, selected, false)

	assert.Equal(t, 1, result.Removed)
	assert.Equal(t, 2, result.Skipped)
	assert.Equal(t, 0, result.Failed)

	repo1Exists, _ := afero.DirExists(fs, "repos/work/repo1")
	repo2Exists, _ := afero.DirExists(fs, "repos/work/repo2")
	assert.True(t, repo1Exists)
	assert.False(t, repo2Exists)
}

func TestPrintUncloneSummary_NotFound(t *testing.T) {
	var buf bytes.Buffer
	r := uncloneResult{Removed: 1, Skipped: 0, Failed: 0}

	printUncloneSummary(&buf, r, []string{"missing-repo"})

	out := buf.String()
	assert.Contains(t, out, `warning: "missing-repo" not found`)
	assert.Contains(t, out, "Summary: 1 uncloned, 0 skipped, 0 failed")
}

func TestPrintUncloneSummary_Counts(t *testing.T) {
	var buf bytes.Buffer
	r := uncloneResult{Removed: 2, Skipped: 3, Failed: 1}

	printUncloneSummary(&buf, r, nil)

	assert.True(t, strings.Contains(buf.String(), "Summary: 2 uncloned, 3 skipped, 1 failed"))
}
