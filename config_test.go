package main

import (
	"encoding/json"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func baseConfig() map[string]any {
	return map[string]any{
		"profiles": map[string]any{
			"personal": map[string]any{
				"user_name":  "Jane Doe",
				"user_email": "jane.doe@example.com",
				"ssh_key":    "~/.ssh/id_rsa",
			},
			"work": map[string]any{
				"user_name":  "Jane Doe",
				"user_email": "jane.doe@work-example.com",
				"ssh_key":    "~/.ssh/id_rsa_work",
			},
		},
		"platforms": map[string]any{
			"company": map[string]any{"host": "git.company.com"},
		},
		"default_profile":  "personal",
		"default_platform": "github",
		"repositories": map[string]any{
			"work/dynamic-routing": []any{
				map[string]any{
					"repository": "company/active-task-pool",
					"name":       "active-task-pool",
					"aliases":    []any{"ATP"},
					"platform":   "company",
					"profile":    "work",
				},
				map[string]any{
					"repository": "company/task-pool",
					"name":       "task-pool",
					"aliases":    []any{"TP"},
					"profile":    "work",
				},
			},
			"personal/static-routing": []any{
				map[string]any{
					"repository": "example-user/plan-assignment",
					"name":       "plan-assignment",
				},
			},
		},
	}
}

func repoEntry(repository, name string) map[string]any {
	return map[string]any{"repository": repository, "name": name}
}

func repositoriesOf(config map[string]any) map[string]any {
	return config["repositories"].(map[string]any)
}

func workRepoOf(config map[string]any, index int) map[string]any {
	return repositoriesOf(config)["work/dynamic-routing"].([]any)[index].(map[string]any)
}

func writeConfig(t *testing.T, config map[string]any) afero.Fs {
	t.Helper()

	data, err := json.Marshal(config)
	require.NoError(t, err)

	fs := afero.NewMemMapFs()
	require.NoError(t, afero.WriteFile(fs, configPath, data, 0o644))

	return fs
}

func writeLocalConfig(t *testing.T, fs afero.Fs, local map[string]any) {
	t.Helper()

	data, err := json.Marshal(local)
	require.NoError(t, err)

	require.NoError(t, afero.WriteFile(fs, localConfigPath, data, 0o644))
}

func TestNewConfig(t *testing.T) {
	config, err := NewConfig(writeConfig(t, baseConfig()))
	require.NoError(t, err)

	assert.Equal(t, "github.com", config.Platforms["github"].Host)
	assert.Equal(t, "gitlab.com", config.Platforms["gitlab"].Host)
	assert.Equal(t, "bitbucket.org", config.Platforms["bitbucket"].Host)
	assert.Equal(t, "git.company.com", config.Platforms["company"].Host)

	work := config.Repositories["work/dynamic-routing"]
	require.Len(t, work, 2)
	assert.Equal(t, "company/active-task-pool", work[0].Repository)
	assert.Equal(t, "active-task-pool", work[0].Name)
	assert.Equal(t, []string{"ATP"}, work[0].Aliases)
	assert.Equal(t, "company", work[0].Platform)
	assert.Equal(t, "work", work[0].Profile)

	personal := config.Repositories["personal/static-routing"]
	require.Len(t, personal, 1)
	assert.Empty(t, personal[0].Platform)
	assert.Empty(t, personal[0].Aliases)
}

func TestNewConfigMissingFile(t *testing.T) {
	_, err := NewConfig(afero.NewMemMapFs())
	assert.Error(t, err)
}

func TestNewConfigInvalidJSON(t *testing.T) {
	fs := afero.NewMemMapFs()
	require.NoError(t, afero.WriteFile(fs, configPath, []byte("{invalid"), 0o644))

	_, err := NewConfig(fs)
	assert.Error(t, err)
}

func TestNewConfigKeepsUserDefinedDefaultPlatform(t *testing.T) {
	config := baseConfig()
	config["platforms"].(map[string]any)["github"] = map[string]any{"host": "github.enterprise.com"}

	loaded, err := NewConfig(writeConfig(t, config))
	require.NoError(t, err)

	assert.Equal(t, "github.enterprise.com", loaded.Platforms["github"].Host)
}

func TestNewConfigLocalDeepMerge(t *testing.T) {
	fs := writeConfig(t, baseConfig())
	writeLocalConfig(t, fs, map[string]any{
		"default_profile": "work",
		"profiles": map[string]any{
			"personal": map[string]any{"ssh_key": "/other/key"},
		},
	})

	config, err := NewConfig(fs)
	require.NoError(t, err)

	assert.Equal(t, "work", config.DefaultProfile)
	assert.Equal(t, "/other/key", config.Profiles["personal"].SSHKey)
	assert.Equal(t, "Jane Doe", config.Profiles["personal"].UserName)
	assert.Equal(t, "jane.doe@example.com", config.Profiles["personal"].UserEmail)
	assert.Equal(t, "~/.ssh/id_rsa_work", config.Profiles["work"].SSHKey)
}

func TestNewConfigLocalReplacesRepositoryArrays(t *testing.T) {
	fs := writeConfig(t, baseConfig())
	writeLocalConfig(t, fs, map[string]any{
		"repositories": map[string]any{
			"work/dynamic-routing": []any{
				map[string]any{
					"repository": "company/other",
					"name":       "other",
					"platform":   "company",
					"profile":    "work",
				},
			},
		},
	})

	config, err := NewConfig(fs)
	require.NoError(t, err)

	work := config.Repositories["work/dynamic-routing"]
	require.Len(t, work, 1)
	assert.Equal(t, "company/other", work[0].Repository)
	assert.Len(t, config.Repositories["personal/static-routing"], 1)
}

func TestNewConfigLocalRemovesWithNull(t *testing.T) {
	fs := writeConfig(t, baseConfig())
	writeLocalConfig(t, fs, map[string]any{
		"repositories": map[string]any{"personal/static-routing": nil},
	})

	config, err := NewConfig(fs)
	require.NoError(t, err)

	assert.NotContains(t, config.Repositories, "personal/static-routing")
	assert.Contains(t, config.Repositories, "work/dynamic-routing")
}

func TestNewConfigLocalOverridesBuiltInPlatformHost(t *testing.T) {
	fs := writeConfig(t, baseConfig())
	writeLocalConfig(t, fs, map[string]any{
		"platforms": map[string]any{"github": map[string]any{"host": "github.enterprise.com"}},
	})

	config, err := NewConfig(fs)
	require.NoError(t, err)

	assert.Equal(t, "github.enterprise.com", config.Platforms["github"].Host)
}

func TestNewConfigLocalNullRestoresBuiltInPlatform(t *testing.T) {
	config := baseConfig()
	config["platforms"].(map[string]any)["github"] = map[string]any{"host": "github.enterprise.com"}

	fs := writeConfig(t, config)
	writeLocalConfig(t, fs, map[string]any{
		"platforms": map[string]any{"github": nil},
	})

	loaded, err := NewConfig(fs)
	require.NoError(t, err)

	assert.Equal(t, "github.com", loaded.Platforms["github"].Host)
}

func TestNewConfigLocalCompletesBase(t *testing.T) {
	config := baseConfig()
	profiles := config["profiles"]
	delete(config, "profiles")

	fs := writeConfig(t, config)
	writeLocalConfig(t, fs, map[string]any{"profiles": profiles})

	loaded, err := NewConfig(fs)
	require.NoError(t, err)

	assert.Equal(t, "~/.ssh/id_rsa", loaded.Profiles["personal"].SSHKey)
	assert.Equal(t, "~/.ssh/id_rsa_work", loaded.Profiles["work"].SSHKey)
}

func TestNewConfigLocalEmptyObjectIsNoOp(t *testing.T) {
	fs := writeConfig(t, baseConfig())
	writeLocalConfig(t, fs, map[string]any{})

	config, err := NewConfig(fs)
	require.NoError(t, err)

	assert.Equal(t, "git.company.com", config.Platforms["company"].Host)
	assert.Len(t, config.Repositories, 2)
}

func TestNewConfigLocalInvalidJSON(t *testing.T) {
	fs := writeConfig(t, baseConfig())
	require.NoError(t, afero.WriteFile(fs, localConfigPath, []byte("{invalid"), 0o644))

	_, err := NewConfig(fs)
	assert.ErrorContains(t, err, localConfigPath)
}

func TestNewConfigInvalidJSONWithLocalPresent(t *testing.T) {
	fs := afero.NewMemMapFs()
	require.NoError(t, afero.WriteFile(fs, configPath, []byte("{invalid"), 0o644))
	writeLocalConfig(t, fs, map[string]any{})

	_, err := NewConfig(fs)
	require.Error(t, err)
	assert.ErrorContains(t, err, configPath)
	assert.NotContains(t, err.Error(), localConfigPath)
}

func TestNewConfigLocalNotAnObject(t *testing.T) {
	for _, doc := range []string{`[1, 2]`, `"text"`, `null`, `3`} {
		t.Run(doc, func(t *testing.T) {
			fs := writeConfig(t, baseConfig())
			require.NoError(t, afero.WriteFile(fs, localConfigPath, []byte(doc), 0o644))

			_, err := NewConfig(fs)
			assert.ErrorContains(t, err, localConfigPath)
			assert.ErrorContains(t, err, "must be a JSON object")
		})
	}
}

func TestNewConfigLocalValidationErrorMentionsLocalFile(t *testing.T) {
	fs := writeConfig(t, baseConfig())
	writeLocalConfig(t, fs, map[string]any{
		"profiles": map[string]any{"work": nil},
	})

	_, err := NewConfig(fs)
	assert.ErrorContains(t, err, "references unknown profile 'work'")
	assert.ErrorContains(t, err, localConfigPath)
}

func TestNewConfigValidationErrors(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(config map[string]any)
		wantErr string
	}{
		{
			name: "absolute directory",
			mutate: func(config map[string]any) {
				repositoriesOf(config)["/abs"] = []any{repoEntry("owner/x", "x")}
			},
			wantErr: "must be a clean relative path",
		},
		{
			name: "directory escaping the hub",
			mutate: func(config map[string]any) {
				repositoriesOf(config)["../out"] = []any{repoEntry("owner/x", "x")}
			},
			wantErr: "must be a clean relative path",
		},
		{
			name: "directory not clean",
			mutate: func(config map[string]any) {
				repositoriesOf(config)["work//nested"] = []any{repoEntry("owner/x", "x")}
			},
			wantErr: "must be a clean relative path",
		},
		{
			name: "empty directory",
			mutate: func(config map[string]any) {
				repositoriesOf(config)[""] = []any{repoEntry("owner/x", "x")}
			},
			wantErr: "must be a clean relative path",
		},
		{
			name: "directory without repositories",
			mutate: func(config map[string]any) {
				repositoriesOf(config)["empty/dir"] = []any{}
			},
			wantErr: "has no repositories",
		},
		{
			name: "null directory",
			mutate: func(config map[string]any) {
				repositoriesOf(config)["null/dir"] = nil
			},
			wantErr: "has no repositories",
		},
		{
			name: "repository without owner",
			mutate: func(config map[string]any) {
				workRepoOf(config, 0)["repository"] = "task-pool"
			},
			wantErr: "must have the format",
		},
		{
			name: "repository with empty segment",
			mutate: func(config map[string]any) {
				workRepoOf(config, 0)["repository"] = "company/"
			},
			wantErr: "must have the format",
		},
		{
			name: "empty name",
			mutate: func(config map[string]any) {
				workRepoOf(config, 0)["name"] = ""
			},
			wantErr: "must have a valid directory name",
		},
		{
			name: "name with slash",
			mutate: func(config map[string]any) {
				workRepoOf(config, 0)["name"] = "a/b"
			},
			wantErr: "must have a valid directory name",
		},
		{
			name: "empty alias",
			mutate: func(config map[string]any) {
				workRepoOf(config, 0)["aliases"] = []any{""}
			},
			wantErr: "has an empty alias",
		},
		{
			name: "duplicate name",
			mutate: func(config map[string]any) {
				workRepoOf(config, 0)["name"] = "task-pool"
			},
			wantErr: "name or alias 'task-pool' is used by more than one repository",
		},
		{
			name: "duplicate alias",
			mutate: func(config map[string]any) {
				workRepoOf(config, 0)["aliases"] = []any{"TP"}
			},
			wantErr: "name or alias 'TP' is used by more than one repository",
		},
		{
			name: "alias equal to another name",
			mutate: func(config map[string]any) {
				workRepoOf(config, 0)["aliases"] = []any{"plan-assignment"}
			},
			wantErr: "name or alias 'plan-assignment' is used by more than one repository",
		},
		{
			name: "duplicate repository",
			mutate: func(config map[string]any) {
				workRepoOf(config, 0)["repository"] = "company/task-pool"
			},
			wantErr: "repository 'company/task-pool' is defined more than once",
		},
		{
			name: "no platform and no default platform",
			mutate: func(config map[string]any) {
				config["default_platform"] = ""
			},
			wantErr: "has no platform",
		},
		{
			name: "no profile and no default profile",
			mutate: func(config map[string]any) {
				config["default_profile"] = ""
			},
			wantErr: "has no profile",
		},
		{
			name: "unknown platform",
			mutate: func(config map[string]any) {
				workRepoOf(config, 0)["platform"] = "unknown"
			},
			wantErr: "references unknown platform 'unknown'",
		},
		{
			name: "unknown profile",
			mutate: func(config map[string]any) {
				workRepoOf(config, 0)["profile"] = "unknown"
			},
			wantErr: "references unknown profile 'unknown'",
		},
		{
			name: "directory inside a clone destination",
			mutate: func(config map[string]any) {
				repositoriesOf(config)["work/dynamic-routing/task-pool/nested"] = []any{repoEntry("owner/x", "x")}
			},
			wantErr: "is inside the clone destination 'work/dynamic-routing/task-pool'",
		},
		{
			name: "directory equal to a clone destination",
			mutate: func(config map[string]any) {
				repositoriesOf(config)["work/dynamic-routing/task-pool"] = []any{repoEntry("owner/x", "x")}
			},
			wantErr: "is inside the clone destination 'work/dynamic-routing/task-pool'",
		},
		{
			name: "default platform does not exist",
			mutate: func(config map[string]any) {
				config["default_platform"] = "unknown"
			},
			wantErr: "default platform 'unknown' does not exist",
		},
		{
			name: "default profile does not exist",
			mutate: func(config map[string]any) {
				config["default_profile"] = "unknown"
			},
			wantErr: "default profile 'unknown' does not exist",
		},
		{
			name: "empty profiles",
			mutate: func(config map[string]any) {
				config["profiles"] = map[string]any{}
			},
			wantErr: "profiles cannot be empty",
		},
		{
			name: "invalid profile email",
			mutate: func(config map[string]any) {
				config["profiles"].(map[string]any)["personal"].(map[string]any)["user_email"] = "not-an-email"
			},
			wantErr: "invalid email address",
		},
		{
			name: "empty profile ssh key",
			mutate: func(config map[string]any) {
				config["profiles"].(map[string]any)["personal"].(map[string]any)["ssh_key"] = ""
			},
			wantErr: "has an empty SSH key",
		},
		{
			name: "platform with empty host",
			mutate: func(config map[string]any) {
				config["platforms"].(map[string]any)["company"] = map[string]any{"host": ""}
			},
			wantErr: "has an empty host",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := baseConfig()
			tt.mutate(config)

			_, err := NewConfig(writeConfig(t, config))
			assert.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestRepositoriesAllIsSortedByDirectory(t *testing.T) {
	repositories := Repositories{
		"b/dir": {{Repository: "owner/r2", Name: "r2"}},
		"a/dir": {{Repository: "owner/r1", Name: "r1"}, {Repository: "owner/r3", Name: "r3"}},
	}

	var visited []string
	for dir, repo := range repositories.All() {
		visited = append(visited, dir+":"+repo.Name)
	}

	assert.Equal(t, []string{"a/dir:r1", "a/dir:r3", "b/dir:r2"}, visited)
}
