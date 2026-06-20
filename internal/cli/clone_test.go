package cli

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/willpinha/yhub/internal/config"
	"github.com/willpinha/yhub/internal/git/gittest"
)

func TestSelectAll_SortedGroupOrder(t *testing.T) {
	cfg := &config.Config{
		Groups: map[string][]config.Repository{
			"z-group": {{Alias: "ZR", Name: "z-repo", Repository: "org/z-repo"}},
			"a-group": {{Alias: "AR", Name: "a-repo", Repository: "org/a-repo"}},
			"m-group": {{Alias: "MR", Name: "m-repo", Repository: "org/m-repo"}},
		},
	}

	result := selectAll(cfg)

	require.Len(t, result, 3)
	assert.Equal(t, "a-group", result[0].Group)
	assert.Equal(t, "m-group", result[1].Group)
	assert.Equal(t, "z-group", result[2].Group)
}

func TestSelectAll_AllReposIncluded(t *testing.T) {
	cfg := &config.Config{
		Groups: map[string][]config.Repository{
			"g1": {
				{Alias: "R1", Name: "repo1", Repository: "org/repo1"},
				{Alias: "R2", Name: "repo2", Repository: "org/repo2"},
			},
			"g2": {
				{Alias: "R3", Name: "repo3", Repository: "org/repo3"},
			},
		},
	}

	result := selectAll(cfg)

	require.Len(t, result, 3)
}

func TestSelectGroups_Found(t *testing.T) {
	cfg := &config.Config{
		Groups: map[string][]config.Repository{
			"work": {
				{Alias: "WR", Name: "work-repo", Repository: "org/work-repo"},
			},
			"personal": {
				{Alias: "PR", Name: "personal-repo", Repository: "org/personal-repo"},
			},
		},
	}

	selected, notFound := selectGroups(cfg, []string{"work"})

	require.Len(t, selected, 1)
	assert.Equal(t, "work", selected[0].Group)
	assert.Equal(t, "work-repo", selected[0].Repo.Name)
	assert.Empty(t, notFound)
}

func TestSelectGroups_NotFound(t *testing.T) {
	cfg := &config.Config{
		Groups: map[string][]config.Repository{
			"work": {{Alias: "WR", Name: "work-repo", Repository: "org/work-repo"}},
		},
	}

	selected, notFound := selectGroups(cfg, []string{"work", "missing"})

	require.Len(t, selected, 1)
	require.Len(t, notFound, 1)
	assert.Equal(t, "missing", notFound[0])
}

func TestSelectRepos_FoundByAlias(t *testing.T) {
	cfg := &config.Config{
		Groups: map[string][]config.Repository{
			"work": {
				{Alias: "WR", Name: "work-repo", Repository: "org/work-repo"},
			},
		},
	}

	selected, notFound := selectRepos(cfg, []string{"WR"})

	require.Len(t, selected, 1)
	assert.Equal(t, "work-repo", selected[0].Repo.Name)
	assert.Empty(t, notFound)
}

func TestSelectRepos_FoundByName(t *testing.T) {
	cfg := &config.Config{
		Groups: map[string][]config.Repository{
			"work": {
				{Alias: "WR", Name: "work-repo", Repository: "org/work-repo"},
			},
		},
	}

	selected, notFound := selectRepos(cfg, []string{"work-repo"})

	require.Len(t, selected, 1)
	assert.Equal(t, "work-repo", selected[0].Repo.Name)
	assert.Empty(t, notFound)
}

func TestSelectRepos_MixedNamesAndAliases(t *testing.T) {
	cfg := &config.Config{
		Groups: map[string][]config.Repository{
			"work": {
				{Alias: "WR", Name: "work-repo", Repository: "org/work-repo"},
				{Alias: "PR", Name: "personal-repo", Repository: "org/personal-repo"},
			},
		},
	}

	selected, notFound := selectRepos(cfg, []string{"WR", "personal-repo"})

	require.Len(t, selected, 2)
	names := []string{selected[0].Repo.Name, selected[1].Repo.Name}
	assert.Contains(t, names, "work-repo")
	assert.Contains(t, names, "personal-repo")
	assert.Empty(t, notFound)
}

func TestSelectRepos_DedupNameAndAlias(t *testing.T) {
	cfg := &config.Config{
		Groups: map[string][]config.Repository{
			"work": {
				{Alias: "WR", Name: "work-repo", Repository: "org/work-repo"},
			},
		},
	}

	selected, notFound := selectRepos(cfg, []string{"WR", "work-repo"})

	require.Len(t, selected, 1)
	assert.Equal(t, "work-repo", selected[0].Repo.Name)
	assert.Empty(t, notFound)
}

func TestSelectRepos_DedupDuplicateIdent(t *testing.T) {
	cfg := &config.Config{
		Groups: map[string][]config.Repository{
			"work": {
				{Alias: "WR", Name: "work-repo", Repository: "org/work-repo"},
			},
		},
	}

	selected, notFound := selectRepos(cfg, []string{"WR", "WR"})

	require.Len(t, selected, 1)
	assert.Empty(t, notFound)
}

func TestSelectRepos_NotFound(t *testing.T) {
	cfg := &config.Config{
		Groups: map[string][]config.Repository{
			"work": {
				{Alias: "WR", Name: "work-repo", Repository: "org/work-repo"},
			},
		},
	}

	selected, notFound := selectRepos(cfg, []string{"WR", "MISSING"})

	require.Len(t, selected, 1)
	require.Len(t, notFound, 1)
	assert.Equal(t, "MISSING", notFound[0])
}

func TestSelectRepos_AliasTakesPrecedenceOverName(t *testing.T) {
	cfg := &config.Config{
		Groups: map[string][]config.Repository{
			"a-group": {
				{Alias: "shared-ident", Name: "alias-repo", Repository: "org/alias-repo"},
			},
			"b-group": {
				{Alias: "BR", Name: "shared-ident", Repository: "org/name-repo"},
			},
		},
	}

	selected, notFound := selectRepos(cfg, []string{"shared-ident"})

	require.Len(t, selected, 1)
	assert.Equal(t, "alias-repo", selected[0].Repo.Name)
	assert.Empty(t, notFound)
}

func TestSelectRepos_DeterministicOrder(t *testing.T) {
	cfg := &config.Config{
		Groups: map[string][]config.Repository{
			"z-group": {{Alias: "ZR", Name: "z-repo", Repository: "org/z-repo"}},
			"a-group": {{Alias: "AR", Name: "a-repo", Repository: "org/a-repo"}},
		},
	}

	selected, _ := selectRepos(cfg, []string{"z-repo", "a-repo"})

	require.Len(t, selected, 2)
	assert.Equal(t, "z-repo", selected[0].Repo.Name)
	assert.Equal(t, "a-repo", selected[1].Repo.Name)
}

func TestRepoDest(t *testing.T) {
	cfg := &config.Config{RepositoriesDir: "/home/user/repos"}
	s := selectedRepo{Group: "work", Repo: config.Repository{Name: "my-repo"}}

	dest := repoDest(cfg, s)

	assert.Equal(t, filepath.Join("/home/user/repos", "work", "my-repo"), dest)
}

func TestExpandHome_TildeExpands(t *testing.T) {
	home, err := os.UserHomeDir()
	require.NoError(t, err)

	result := expandHome("~/foo/bar")

	assert.Equal(t, filepath.Join(home, "foo/bar"), result)
}

func TestExpandHome_NonTildeUnchanged(t *testing.T) {
	result := expandHome("/absolute/path")

	assert.Equal(t, "/absolute/path", result)
}

func TestFindOrphans_NoOrphans(t *testing.T) {
	fs := afero.NewMemMapFs()
	cfg := &config.Config{
		RepositoriesDir: "repos",
		Groups: map[string][]config.Repository{
			"work": {{Name: "repo1"}},
		},
	}
	require.NoError(t, fs.MkdirAll("repos/work/repo1", 0755))

	orphans, err := findOrphans(fs, cfg)

	require.NoError(t, err)
	assert.Empty(t, orphans)
}

func TestFindOrphans_ExtraDir(t *testing.T) {
	fs := afero.NewMemMapFs()
	cfg := &config.Config{
		RepositoriesDir: "repos",
		Groups: map[string][]config.Repository{
			"work": {{Name: "repo1"}},
		},
	}
	require.NoError(t, fs.MkdirAll("repos/work/repo1", 0755))
	require.NoError(t, fs.MkdirAll("repos/work/undeclared", 0755))

	orphans, err := findOrphans(fs, cfg)

	require.NoError(t, err)
	require.Len(t, orphans, 1)
	assert.Equal(t, filepath.Join("work", "undeclared"), orphans[0])
}

func TestFindOrphans_MissingReposDir(t *testing.T) {
	fs := afero.NewMemMapFs()
	cfg := &config.Config{
		RepositoriesDir: "repos",
		Groups:          map[string][]config.Repository{},
	}

	orphans, err := findOrphans(fs, cfg)

	require.NoError(t, err)
	assert.Nil(t, orphans)
}

func TestRunClone_HappyPath(t *testing.T) {
	fs := afero.NewMemMapFs()
	g := &gittest.FakeGit{}
	cfg := &config.Config{
		RepositoriesDir: "repos",
		Groups: map[string][]config.Repository{
			"work": {{Alias: "WR", Name: "repo1", Repository: "org/repo1"}},
		},
	}

	selected := []selectedRepo{{Group: "work", Repo: cfg.Groups["work"][0]}}
	var buf bytes.Buffer

	result := runClone(context.Background(), &buf, fs, g, cfg, nil, selected)

	assert.Equal(t, 1, result.Cloned)
	assert.Equal(t, 0, result.Skipped)
	assert.Equal(t, 0, result.Failed)
	require.Len(t, g.Clones, 1)
	assert.Contains(t, g.Clones[0].URL, "org/repo1")
}

func TestRunClone_Idempotency(t *testing.T) {
	fs := afero.NewMemMapFs()
	g := &gittest.FakeGit{}
	cfg := &config.Config{
		RepositoriesDir: "repos",
		Groups: map[string][]config.Repository{
			"work": {{Alias: "WR", Name: "repo1", Repository: "org/repo1"}},
		},
	}
	require.NoError(t, fs.MkdirAll("repos/work/repo1", 0755))

	selected := []selectedRepo{{Group: "work", Repo: cfg.Groups["work"][0]}}
	var buf bytes.Buffer

	result := runClone(context.Background(), &buf, fs, g, cfg, nil, selected)

	assert.Equal(t, 0, result.Cloned)
	assert.Equal(t, 1, result.Skipped)
	assert.Empty(t, g.Clones)
}

func TestRunClone_SSHEnv(t *testing.T) {
	fs := afero.NewMemMapFs()
	g := &gittest.FakeGit{}
	cfg := &config.Config{
		RepositoriesDir: "repos",
		DefaultProtocol: "ssh",
		Groups: map[string][]config.Repository{
			"work": {{Alias: "WR", Name: "repo1", Repository: "org/repo1", Profile: "work"}},
		},
	}
	local := &config.LocalConfig{
		Profiles: map[string]config.Profile{
			"work": {SSHKey: "/home/user/.ssh/id_work"},
		},
	}

	selected := []selectedRepo{{Group: "work", Repo: cfg.Groups["work"][0]}}
	var buf bytes.Buffer

	runClone(context.Background(), &buf, fs, g, cfg, local, selected)

	require.Len(t, g.Clones, 1)
	require.Len(t, g.Clones[0].Env, 1)
	assert.True(t, strings.HasPrefix(g.Clones[0].Env[0], "GIT_SSH_COMMAND=ssh -i"))
	assert.Contains(t, g.Clones[0].Env[0], "/home/user/.ssh/id_work")
}

func TestRunClone_HTTPSNoEnv(t *testing.T) {
	fs := afero.NewMemMapFs()
	g := &gittest.FakeGit{}
	cfg := &config.Config{
		RepositoriesDir: "repos",
		DefaultProtocol: "https",
		Groups: map[string][]config.Repository{
			"work": {{Alias: "WR", Name: "repo1", Repository: "org/repo1", Profile: "work"}},
		},
	}
	local := &config.LocalConfig{
		Profiles: map[string]config.Profile{
			"work": {SSHKey: "/home/user/.ssh/id_work"},
		},
	}

	selected := []selectedRepo{{Group: "work", Repo: cfg.Groups["work"][0]}}
	var buf bytes.Buffer

	runClone(context.Background(), &buf, fs, g, cfg, local, selected)

	require.Len(t, g.Clones, 1)
	assert.Nil(t, g.Clones[0].Env)
}

func TestRunClone_PostCloneIdentity(t *testing.T) {
	fs := afero.NewMemMapFs()
	g := &gittest.FakeGit{}
	cfg := &config.Config{
		RepositoriesDir: "repos",
		Groups: map[string][]config.Repository{
			"work": {{Alias: "WR", Name: "repo1", Repository: "org/repo1", Profile: "work"}},
		},
	}
	local := &config.LocalConfig{
		Profiles: map[string]config.Profile{
			"work": {Name: "Alice", Email: "alice@example.com"},
		},
	}

	selected := []selectedRepo{{Group: "work", Repo: cfg.Groups["work"][0]}}
	var buf bytes.Buffer

	result := runClone(context.Background(), &buf, fs, g, cfg, local, selected)

	assert.Equal(t, 1, result.Cloned)
	assert.Empty(t, result.Warnings)
	require.Len(t, g.SetConfigs, 2)

	keys := map[string]string{}
	for _, sc := range g.SetConfigs {
		keys[sc.Key] = sc.Value
	}
	assert.Equal(t, "Alice", keys["user.name"])
	assert.Equal(t, "alice@example.com", keys["user.email"])
}

func TestRunClone_PostCloneIdentitySSHKey(t *testing.T) {
	home, err := os.UserHomeDir()
	require.NoError(t, err)

	fs := afero.NewMemMapFs()
	g := &gittest.FakeGit{}
	cfg := &config.Config{
		RepositoriesDir: "repos",
		DefaultProtocol: "ssh",
		Groups: map[string][]config.Repository{
			"work": {{Alias: "WR", Name: "repo1", Repository: "org/repo1", Profile: "work"}},
		},
	}
	local := &config.LocalConfig{
		Profiles: map[string]config.Profile{
			"work": {Name: "Alice", Email: "alice@example.com", SSHKey: "~/.ssh/id_work"},
		},
	}

	selected := []selectedRepo{{Group: "work", Repo: cfg.Groups["work"][0]}}
	var buf bytes.Buffer

	result := runClone(context.Background(), &buf, fs, g, cfg, local, selected)

	assert.Equal(t, 1, result.Cloned)
	assert.Empty(t, result.Warnings)
	require.Len(t, g.SetConfigs, 3)

	vals := map[string]string{}
	for _, sc := range g.SetConfigs {
		vals[sc.Key] = sc.Value
	}
	assert.Equal(t, "Alice", vals["user.name"])
	assert.Equal(t, "alice@example.com", vals["user.email"])
	assert.Equal(t, "ssh -i "+filepath.Join(home, ".ssh/id_work")+" -o IdentitiesOnly=yes", vals["core.sshCommand"])
}

func TestRunClone_PostCloneNoSSHCommandForHTTPS(t *testing.T) {
	fs := afero.NewMemMapFs()
	g := &gittest.FakeGit{}
	cfg := &config.Config{
		RepositoriesDir: "repos",
		DefaultProtocol: "https",
		Groups: map[string][]config.Repository{
			"work": {{Alias: "WR", Name: "repo1", Repository: "org/repo1", Profile: "work"}},
		},
	}
	local := &config.LocalConfig{
		Profiles: map[string]config.Profile{
			"work": {Name: "Alice", Email: "alice@example.com", SSHKey: "~/.ssh/id_work"},
		},
	}

	selected := []selectedRepo{{Group: "work", Repo: cfg.Groups["work"][0]}}
	var buf bytes.Buffer

	result := runClone(context.Background(), &buf, fs, g, cfg, local, selected)

	assert.Equal(t, 1, result.Cloned)
	assert.Empty(t, result.Warnings)

	for _, sc := range g.SetConfigs {
		assert.NotEqual(t, "core.sshCommand", sc.Key)
	}
}

func TestRunClone_PostCloneNoSSHCommandWhenNoKey(t *testing.T) {
	fs := afero.NewMemMapFs()
	g := &gittest.FakeGit{}
	cfg := &config.Config{
		RepositoriesDir: "repos",
		DefaultProtocol: "ssh",
		Groups: map[string][]config.Repository{
			"work": {{Alias: "WR", Name: "repo1", Repository: "org/repo1", Profile: "work"}},
		},
	}
	local := &config.LocalConfig{
		Profiles: map[string]config.Profile{
			"work": {Name: "Alice", Email: "alice@example.com"},
		},
	}

	selected := []selectedRepo{{Group: "work", Repo: cfg.Groups["work"][0]}}
	var buf bytes.Buffer

	result := runClone(context.Background(), &buf, fs, g, cfg, local, selected)

	assert.Equal(t, 1, result.Cloned)
	assert.Empty(t, result.Warnings)

	for _, sc := range g.SetConfigs {
		assert.NotEqual(t, "core.sshCommand", sc.Key)
	}
}

func TestRunClone_PartialFailure(t *testing.T) {
	fs := afero.NewMemMapFs()
	cloneErr := errors.New("network error")
	callCount := 0
	g := &gittest.FakeGit{
		CloneFunc: func(_ context.Context, _, _ string, _ []string) error {
			callCount++
			if callCount == 1 {
				return cloneErr
			}
			return nil
		},
	}
	cfg := &config.Config{
		RepositoriesDir: "repos",
		Groups: map[string][]config.Repository{
			"work": {
				{Alias: "R1", Name: "repo1", Repository: "org/repo1"},
				{Alias: "R2", Name: "repo2", Repository: "org/repo2"},
			},
		},
	}

	selected := []selectedRepo{
		{Group: "work", Repo: cfg.Groups["work"][0]},
		{Group: "work", Repo: cfg.Groups["work"][1]},
	}
	var buf bytes.Buffer

	result := runClone(context.Background(), &buf, fs, g, cfg, nil, selected)

	assert.Equal(t, 1, result.Cloned)
	assert.Equal(t, 1, result.Failed)
	assert.Equal(t, 0, result.Skipped)
}
