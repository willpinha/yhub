package config

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolvePlatform(t *testing.T) {
	tests := []struct {
		name     string
		cfg      *Config
		repo     Repository
		expected string
	}{
		{
			name:     "repo platform overrides cfg default",
			cfg:      &Config{DefaultPlatform: "github"},
			repo:     Repository{Platform: "gitlab"},
			expected: "gitlab",
		},
		{
			name:     "cfg default used when repo platform empty",
			cfg:      &Config{DefaultPlatform: "bitbucket"},
			repo:     Repository{},
			expected: "bitbucket",
		},
		{
			name:     "builtin default when both empty",
			cfg:      &Config{},
			repo:     Repository{},
			expected: "github",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.cfg.ResolvePlatform(tt.repo))
		})
	}
}

func TestResolveProtocol(t *testing.T) {
	tests := []struct {
		name     string
		cfg      *Config
		repo     Repository
		expected string
	}{
		{
			name:     "repo protocol overrides cfg default",
			cfg:      &Config{DefaultProtocol: "https"},
			repo:     Repository{Protocol: "ssh"},
			expected: "ssh",
		},
		{
			name:     "cfg default used when repo protocol empty",
			cfg:      &Config{DefaultProtocol: "ssh"},
			repo:     Repository{},
			expected: "ssh",
		},
		{
			name:     "builtin default when both empty",
			cfg:      &Config{},
			repo:     Repository{},
			expected: "https",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.cfg.ResolveProtocol(tt.repo))
		})
	}
}

func TestCloneURL(t *testing.T) {
	tests := []struct {
		name        string
		cfg         *Config
		repo        Repository
		expectedURL string
		expectedErr error
	}{
		{
			name:        "all defaults: https + github",
			cfg:         &Config{},
			repo:        Repository{Repository: "owner/repo"},
			expectedURL: "https://github.com/owner/repo.git",
		},
		{
			name:        "ssh + github",
			cfg:         &Config{DefaultProtocol: "ssh"},
			repo:        Repository{Repository: "owner/repo"},
			expectedURL: "git@github.com:owner/repo.git",
		},
		{
			name:        "https + gitlab",
			cfg:         &Config{DefaultPlatform: "gitlab"},
			repo:        Repository{Repository: "owner/repo"},
			expectedURL: "https://gitlab.com/owner/repo.git",
		},
		{
			name:        "ssh + bitbucket",
			cfg:         &Config{DefaultPlatform: "bitbucket", DefaultProtocol: "ssh"},
			repo:        Repository{Repository: "owner/repo"},
			expectedURL: "git@bitbucket.org:owner/repo.git",
		},
		{
			name: "custom platform from cfg.Platforms",
			cfg: &Config{
				Platforms: map[string]Platform{
					"company": {Host: "git.company.com"},
				},
				DefaultPlatform: "company",
			},
			repo:        Repository{Repository: "owner/repo"},
			expectedURL: "https://git.company.com/owner/repo.git",
		},
		{
			name: "per-repo platform overrides cfg default",
			cfg:  &Config{DefaultPlatform: "github"},
			repo: Repository{
				Repository: "owner/repo",
				Platform:   "gitlab",
			},
			expectedURL: "https://gitlab.com/owner/repo.git",
		},
		{
			name: "per-repo protocol overrides cfg default",
			cfg:  &Config{DefaultProtocol: "https"},
			repo: Repository{
				Repository: "owner/repo",
				Protocol:   "ssh",
			},
			expectedURL: "git@github.com:owner/repo.git",
		},
		{
			name:        "unknown platform with no host: ErrUnresolvablePlatform",
			cfg:         &Config{},
			repo:        Repository{Repository: "owner/repo", Platform: "unknown"},
			expectedErr: ErrUnresolvablePlatform,
		},
		{
			name:        "empty repository: ErrUnresolvableRepository",
			cfg:         &Config{},
			repo:        Repository{Repository: ""},
			expectedErr: ErrUnresolvableRepository,
		},
		{
			name:        "no slash in repository: ErrUnresolvableRepository",
			cfg:         &Config{},
			repo:        Repository{Repository: "noslash"},
			expectedErr: ErrUnresolvableRepository,
		},
		{
			name:        "invalid protocol value: ErrUnresolvableProtocol",
			cfg:         &Config{DefaultProtocol: "ftp"},
			repo:        Repository{Repository: "owner/repo"},
			expectedErr: ErrUnresolvableProtocol,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url, err := tt.cfg.CloneURL(tt.repo)
			if tt.expectedErr != nil {
				require.Error(t, err)
				assert.True(t, errors.Is(err, tt.expectedErr))
				assert.Empty(t, url)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedURL, url)
			}
		})
	}
}
