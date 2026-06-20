package config

import (
	"errors"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeFile(t *testing.T, fs afero.Fs, path, content string) {
	t.Helper()
	err := afero.WriteFile(fs, path, []byte(content), 0644)
	require.NoErrorf(t, err, "failed to write %s", path)
}

func TestLoad(t *testing.T) {
	tests := []struct {
		name  string
		setup func(fs afero.Fs)
		path  string
		check func(t *testing.T, cfg *Config, err error)
	}{
		{
			name: "valid config: parses all fields correctly",
			setup: func(fs afero.Fs) {
				writeFile(t, fs, "yhub.toml", `
repositories_dir = "repos"
profiles = ["personal", "work"]

[[groups.hello]]
profile = "personal"
repository = "willpinha/world"
name = "world"
alias = "WRD"

[[groups.foo]]
profile = "work"
repository = "willpinha/bar"
name = "bar"
alias = "BR"

[[groups.foo]]
repository = "willpinha/baz"
name = "baz"
alias = "BZ"
`)
			},
			path: "yhub.toml",
			check: func(t *testing.T, cfg *Config, err error) {
				require.NoError(t, err)
				assert.Equal(t, "repos", cfg.RepositoriesDir)
				require.Len(t, cfg.Profiles, 2)
				assert.Equal(t, []string{"personal", "work"}, cfg.Profiles)

				hello, ok := cfg.Groups["hello"]
				require.True(t, ok, "missing group 'hello'")
				require.Len(t, hello, 1)
				assert.Equal(t, "personal", hello[0].Profile)
				assert.Equal(t, "willpinha/world", hello[0].Repository)
				assert.Equal(t, "world", hello[0].Name)
				assert.Equal(t, "WRD", hello[0].Alias)

				foo, ok := cfg.Groups["foo"]
				require.True(t, ok, "missing group 'foo'")
				require.Len(t, foo, 2)
				assert.Equal(t, "BR", foo[0].Alias)
				assert.Equal(t, "BZ", foo[1].Alias)
			},
		},
		{
			name: "default applied: missing repositories_dir defaults to 'repositories'",
			setup: func(fs afero.Fs) {
				writeFile(t, fs, "yhub.toml", `
profiles = ["personal"]

[[groups.hello]]
repository = "willpinha/world"
name = "world"
alias = "WRD"
`)
			},
			path: "yhub.toml",
			check: func(t *testing.T, cfg *Config, err error) {
				require.NoError(t, err)
				assert.Equal(t, "repositories", cfg.RepositoriesDir)
			},
		},
		{
			name: "explicit repositories_dir is preserved",
			setup: func(fs afero.Fs) {
				writeFile(t, fs, "yhub.toml", `
repositories_dir = "my-repos"

[[groups.hello]]
repository = "willpinha/world"
name = "world"
alias = "WRD"
`)
			},
			path: "yhub.toml",
			check: func(t *testing.T, cfg *Config, err error) {
				require.NoError(t, err)
				assert.Equal(t, "my-repos", cfg.RepositoriesDir)
			},
		},
		{
			name: "optional profile: repo without profile parses with empty string",
			setup: func(fs afero.Fs) {
				writeFile(t, fs, "yhub.toml", `
[[groups.hello]]
repository = "willpinha/world"
name = "world"
alias = "WRD"
`)
			},
			path: "yhub.toml",
			check: func(t *testing.T, cfg *Config, err error) {
				require.NoError(t, err)
				hello := cfg.Groups["hello"]
				require.NotEmpty(t, hello, "group 'hello' is empty")
				assert.Equal(t, "", hello[0].Profile)
			},
		},
		{
			name: "new fields: parses default_platform, default_protocol, platforms map",
			setup: func(fs afero.Fs) {
				writeFile(t, fs, "yhub.toml", `
repositories_dir = "repos"
default_platform = "github"
default_protocol = "ssh"
profiles = ["personal"]

[platforms.company]
host = "git.company.com"

[[groups.hello]]
profile = "personal"
repository = "willpinha/world"
name = "world"
alias = "WRD"
platform = "gitlab"
protocol = "https"
`)
			},
			path: "yhub.toml",
			check: func(t *testing.T, cfg *Config, err error) {
				require.NoError(t, err)
				assert.Equal(t, "github", cfg.DefaultPlatform)
				assert.Equal(t, "ssh", cfg.DefaultProtocol)

				company, ok := cfg.Platforms["company"]
				require.True(t, ok, "missing platform 'company'")
				assert.Equal(t, "git.company.com", company.Host)

				hello := cfg.Groups["hello"]
				require.Len(t, hello, 1)
				assert.Equal(t, "gitlab", hello[0].Platform)
				assert.Equal(t, "https", hello[0].Protocol)
			},
		},
		{
			name: "per-repo platform and protocol are empty when not set",
			setup: func(fs afero.Fs) {
				writeFile(t, fs, "yhub.toml", `
[[groups.hello]]
repository = "willpinha/world"
name = "world"
alias = "WRD"
`)
			},
			path: "yhub.toml",
			check: func(t *testing.T, cfg *Config, err error) {
				require.NoError(t, err)
				hello := cfg.Groups["hello"]
				require.Len(t, hello, 1)
				assert.Equal(t, "", hello[0].Platform)
				assert.Equal(t, "", hello[0].Protocol)
			},
		},
		{
			name:  "file missing: returns ErrNotFound",
			setup: func(fs afero.Fs) {},
			path:  "nonexistent.toml",
			check: func(t *testing.T, cfg *Config, err error) {
				require.ErrorIs(t, err, ErrNotFound)
				assert.Nil(t, cfg)
			},
		},
		{
			name: "malformed TOML: returns ErrInvalidTOML",
			setup: func(fs afero.Fs) {
				writeFile(t, fs, "yhub.toml", `
repositories_dir = [this is not valid toml
`)
			},
			path: "yhub.toml",
			check: func(t *testing.T, cfg *Config, err error) {
				require.ErrorIs(t, err, ErrInvalidTOML)
				assert.Nil(t, cfg)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := afero.NewMemMapFs()
			tt.setup(fs)
			cfg, err := Load(fs, tt.path)
			tt.check(t, cfg, err)
		})
	}
}

func TestLoadLocal(t *testing.T) {
	tests := []struct {
		name  string
		setup func(fs afero.Fs)
		path  string
		check func(t *testing.T, cfg *LocalConfig, err error)
	}{
		{
			name: "valid local config: parses profiles with all fields",
			setup: func(fs afero.Fs) {
				writeFile(t, fs, "yhub.local.toml", `
[profiles.personal]
name     = "willpinha"
email    = "personal@example.com"
ssh_key  = "~/.ssh/id_personal"

[profiles.work]
name  = "will-work"
email = "will@company.com"
`)
			},
			path: "yhub.local.toml",
			check: func(t *testing.T, cfg *LocalConfig, err error) {
				require.NoError(t, err)

				personal, ok := cfg.Profiles["personal"]
				require.True(t, ok, "missing profile 'personal'")
				assert.Equal(t, "willpinha", personal.Name)
				assert.Equal(t, "personal@example.com", personal.Email)
				assert.Equal(t, "~/.ssh/id_personal", personal.SSHKey)

				work, ok := cfg.Profiles["work"]
				require.True(t, ok, "missing profile 'work'")
				assert.Equal(t, "will-work", work.Name)
				assert.Equal(t, "will@company.com", work.Email)
				assert.Equal(t, "", work.SSHKey)
			},
		},
		{
			name: "profile without ssh_key has empty SSHKey",
			setup: func(fs afero.Fs) {
				writeFile(t, fs, "yhub.local.toml", `
[profiles.personal]
name  = "willpinha"
email = "personal@example.com"
`)
			},
			path: "yhub.local.toml",
			check: func(t *testing.T, cfg *LocalConfig, err error) {
				require.NoError(t, err)
				personal := cfg.Profiles["personal"]
				assert.Equal(t, "", personal.SSHKey)
			},
		},
		{
			name:  "file missing: returns ErrNotFound",
			setup: func(fs afero.Fs) {},
			path:  "yhub.local.toml",
			check: func(t *testing.T, cfg *LocalConfig, err error) {
				require.True(t, errors.Is(err, ErrNotFound))
				assert.Nil(t, cfg)
			},
		},
		{
			name: "malformed TOML: returns ErrInvalidTOML",
			setup: func(fs afero.Fs) {
				writeFile(t, fs, "yhub.local.toml", `
[profiles.personal
name = "willpinha"
`)
			},
			path: "yhub.local.toml",
			check: func(t *testing.T, cfg *LocalConfig, err error) {
				require.True(t, errors.Is(err, ErrInvalidTOML))
				assert.Nil(t, cfg)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := afero.NewMemMapFs()
			tt.setup(fs)
			cfg, err := LoadLocal(fs, tt.path)
			tt.check(t, cfg, err)
		})
	}
}
