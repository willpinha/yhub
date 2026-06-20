package config

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validConfig() *Config {
	return &Config{
		RepositoriesDir: "repositories",
		Profiles:        []string{"personal", "work"},
		Groups: map[string][]Repository{
			"hello": {
				{
					Profile:    "personal",
					Repository: "willpinha/world",
					Name:       "world",
					Alias:      "WRD",
				},
			},
			"foo": {
				{
					Profile:    "work",
					Repository: "willpinha/bar",
					Name:       "bar",
					Alias:      "BR",
				},
				{
					Repository: "willpinha/baz",
					Name:       "baz",
					Alias:      "BZ",
				},
			},
		},
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name       string
		cfg        *Config
		wantErr    bool
		sentinels  []error
		substrings []string
	}{
		{
			name:    "valid config returns nil",
			cfg:     validConfig(),
			wantErr: false,
		},
		{
			name: "empty optional profile does not trigger unknown-profile rule",
			cfg: &Config{
				Profiles: []string{"personal"},
				Groups: map[string][]Repository{
					"g": {
						{Profile: "", Repository: "a/b", Name: "b", Alias: "B"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "empty repository field",
			cfg: &Config{
				Groups: map[string][]Repository{
					"g": {
						{Repository: "", Name: "n", Alias: "A"},
					},
				},
			},
			wantErr:    true,
			sentinels:  []error{ErrEmptyField},
			substrings: []string{`"repository"`, "empty"},
		},
		{
			name: "empty name field",
			cfg: &Config{
				Groups: map[string][]Repository{
					"g": {
						{Repository: "a/b", Name: "", Alias: "A"},
					},
				},
			},
			wantErr:    true,
			sentinels:  []error{ErrEmptyField},
			substrings: []string{`"name"`, "empty"},
		},
		{
			name: "empty alias field",
			cfg: &Config{
				Groups: map[string][]Repository{
					"g": {
						{Repository: "a/b", Name: "n", Alias: ""},
					},
				},
			},
			wantErr:    true,
			sentinels:  []error{ErrEmptyField},
			substrings: []string{`"alias"`, "empty"},
		},
		{
			name: "unknown profile",
			cfg: &Config{
				Profiles: []string{"personal"},
				Groups: map[string][]Repository{
					"foo": {
						{Profile: "work", Repository: "a/b", Name: "n", Alias: "A"},
					},
				},
			},
			wantErr:    true,
			sentinels:  []error{ErrUnknownProfile},
			substrings: []string{"work", "not declared in profiles"},
		},
		{
			name: "duplicate alias across groups",
			cfg: &Config{
				Groups: map[string][]Repository{
					"g1": {
						{Repository: "a/b", Name: "n1", Alias: "DUP"},
					},
					"g2": {
						{Repository: "c/d", Name: "n2", Alias: "DUP"},
					},
				},
			},
			wantErr:    true,
			sentinels:  []error{ErrDuplicateAlias},
			substrings: []string{"DUP"},
		},
		{
			name: "duplicate name across groups",
			cfg: &Config{
				Groups: map[string][]Repository{
					"g1": {
						{Repository: "a/b", Name: "same", Alias: "A1"},
					},
					"g2": {
						{Repository: "c/d", Name: "same", Alias: "A2"},
					},
				},
			},
			wantErr:    true,
			sentinels:  []error{ErrDuplicateName},
			substrings: []string{"same"},
		},
		{
			name: "malformed repository: a/b/c",
			cfg: &Config{
				Groups: map[string][]Repository{
					"g": {
						{Repository: "a/b/c", Name: "n", Alias: "A"},
					},
				},
			},
			wantErr:    true,
			sentinels:  []error{ErrBadRepositoryFormat},
			substrings: []string{"a/b/c", `"owner/repo" format`},
		},
		{
			name: "malformed repository: trailing slash",
			cfg: &Config{
				Groups: map[string][]Repository{
					"g": {
						{Repository: "a/", Name: "n", Alias: "A"},
					},
				},
			},
			wantErr:   true,
			sentinels: []error{ErrBadRepositoryFormat},
		},
		{
			name: "malformed repository: leading slash",
			cfg: &Config{
				Groups: map[string][]Repository{
					"g": {
						{Repository: "/b", Name: "n", Alias: "A"},
					},
				},
			},
			wantErr:   true,
			sentinels: []error{ErrBadRepositoryFormat},
		},
		{
			name: "malformed repository: no slash",
			cfg: &Config{
				Groups: map[string][]Repository{
					"g": {
						{Repository: "no-slash", Name: "n", Alias: "A"},
					},
				},
			},
			wantErr:   true,
			sentinels: []error{ErrBadRepositoryFormat},
		},
		{
			name: "malformed repository: leading space",
			cfg: &Config{
				Groups: map[string][]Repository{
					"g": {
						{Repository: " a/b", Name: "n", Alias: "A"},
					},
				},
			},
			wantErr:   true,
			sentinels: []error{ErrBadRepositoryFormat},
		},
		{
			name: "multiple violations are all reported",
			cfg: &Config{
				Profiles: []string{"personal"},
				Groups: map[string][]Repository{
					"g1": {
						{Profile: "work", Repository: "", Name: "n1", Alias: "A1"},
					},
					"g2": {
						{Repository: "bad//format", Name: "n2", Alias: "A1"},
					},
				},
			},
			wantErr: true,
			sentinels: []error{
				ErrEmptyField,
				ErrUnknownProfile,
				ErrDuplicateAlias,
				ErrBadRepositoryFormat,
			},
		},
		{
			name: "output is deterministic across calls",
			cfg: &Config{
				Groups: map[string][]Repository{
					"alpha": {{Repository: "", Name: "n1", Alias: "A"}},
					"beta":  {{Repository: "", Name: "n2", Alias: "B"}},
					"gamma": {{Repository: "", Name: "n3", Alias: "C"}},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()

			if !tt.wantErr {
				require.NoError(t, err)
				return
			}

			require.Error(t, err)

			for _, sentinel := range tt.sentinels {
				assert.ErrorIs(t, err, sentinel)
			}

			msg := err.Error()
			for _, sub := range tt.substrings {
				assert.True(t, strings.Contains(msg, sub),
					"expected error message to contain %q, got: %s", sub, msg)
			}
		})
	}
}

func TestValidateDeterminism(t *testing.T) {
	cfg := &Config{
		Groups: map[string][]Repository{
			"zzz": {{Repository: "", Name: "n1", Alias: "A"}},
			"aaa": {{Repository: "", Name: "n2", Alias: "B"}},
			"mmm": {{Repository: "", Name: "n3", Alias: "C"}},
		},
	}

	err1 := cfg.Validate()
	err2 := cfg.Validate()

	require.Error(t, err1)
	require.Error(t, err2)
	assert.Equal(t, err1.Error(), err2.Error(), "Validate must return the same message on repeated calls")

	msg := err1.Error()
	posAaa := strings.Index(msg, `"aaa"`)
	posMmm := strings.Index(msg, `"mmm"`)
	posZzz := strings.Index(msg, `"zzz"`)
	assert.True(t, posAaa < posMmm && posMmm < posZzz,
		"groups should appear in sorted order (aaa < mmm < zzz) in error message")
}

func TestValidateProtocols(t *testing.T) {
	tests := []struct {
		name       string
		cfg        *Config
		wantErr    bool
		sentinels  []error
		substrings []string
	}{
		{
			name:    "empty protocol is allowed",
			cfg:     &Config{},
			wantErr: false,
		},
		{
			name:    "https is valid",
			cfg:     &Config{DefaultProtocol: "https"},
			wantErr: false,
		},
		{
			name:    "ssh is valid",
			cfg:     &Config{DefaultProtocol: "ssh"},
			wantErr: false,
		},
		{
			name:       "invalid default_protocol",
			cfg:        &Config{DefaultProtocol: "ftp"},
			wantErr:    true,
			sentinels:  []error{ErrInvalidProtocol},
			substrings: []string{"default_protocol", "ftp", `"https" or "ssh"`},
		},
		{
			name: "invalid repo protocol",
			cfg: &Config{
				Groups: map[string][]Repository{
					"g": {
						{Repository: "a/b", Name: "n", Alias: "A", Protocol: "git"},
					},
				},
			},
			wantErr:    true,
			sentinels:  []error{ErrInvalidProtocol},
			substrings: []string{`"g"`, `"A"`, "git", `"https" or "ssh"`},
		},
		{
			name: "empty repo protocol is allowed",
			cfg: &Config{
				Groups: map[string][]Repository{
					"g": {
						{Repository: "a/b", Name: "n", Alias: "A", Protocol: ""},
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := validateProtocols(tt.cfg)
			err := errors.Join(errs...)

			if !tt.wantErr {
				require.NoError(t, err)
				return
			}

			require.Error(t, err)
			for _, sentinel := range tt.sentinels {
				assert.ErrorIs(t, err, sentinel)
			}
			msg := err.Error()
			for _, sub := range tt.substrings {
				assert.True(t, strings.Contains(msg, sub),
					"expected error message to contain %q, got: %s", sub, msg)
			}
		})
	}
}

func TestValidatePlatforms(t *testing.T) {
	tests := []struct {
		name       string
		cfg        *Config
		wantErr    bool
		sentinels  []error
		substrings []string
	}{
		{
			name:    "empty default_platform is allowed",
			cfg:     &Config{},
			wantErr: false,
		},
		{
			name:    "builtin github is known",
			cfg:     &Config{DefaultPlatform: "github"},
			wantErr: false,
		},
		{
			name:    "builtin gitlab is known",
			cfg:     &Config{DefaultPlatform: "gitlab"},
			wantErr: false,
		},
		{
			name:    "builtin bitbucket is known",
			cfg:     &Config{DefaultPlatform: "bitbucket"},
			wantErr: false,
		},
		{
			name: "custom declared platform is known",
			cfg: &Config{
				DefaultPlatform: "myforge",
				Platforms:       map[string]Platform{"myforge": {Host: "myforge.example.com"}},
			},
			wantErr: false,
		},
		{
			name:       "unknown default_platform",
			cfg:        &Config{DefaultPlatform: "unknown"},
			wantErr:    true,
			sentinels:  []error{ErrUnknownPlatform},
			substrings: []string{"default_platform", "unknown", "not a known platform"},
		},
		{
			name: "unknown repo platform",
			cfg: &Config{
				Groups: map[string][]Repository{
					"g": {
						{Repository: "a/b", Name: "n", Alias: "A", Platform: "nope"},
					},
				},
			},
			wantErr:    true,
			sentinels:  []error{ErrUnknownPlatform},
			substrings: []string{`"g"`, `"A"`, "nope", "not a known platform"},
		},
		{
			name: "platform with empty host",
			cfg: &Config{
				Platforms: map[string]Platform{"myforge": {Host: ""}},
			},
			wantErr:    true,
			sentinels:  []error{ErrEmptyPlatformHost},
			substrings: []string{"myforge", "empty host"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := validatePlatforms(tt.cfg)
			err := errors.Join(errs...)

			if !tt.wantErr {
				require.NoError(t, err)
				return
			}

			require.Error(t, err)
			for _, sentinel := range tt.sentinels {
				assert.ErrorIs(t, err, sentinel)
			}
			msg := err.Error()
			for _, sub := range tt.substrings {
				assert.True(t, strings.Contains(msg, sub),
					"expected error message to contain %q, got: %s", sub, msg)
			}
		})
	}
}

func TestValidateProfiles(t *testing.T) {
	tests := []struct {
		name       string
		cfg        *Config
		local      *LocalConfig
		wantErr    bool
		sentinels  []error
		substrings []string
	}{
		{
			name:    "both empty returns no error",
			cfg:     &Config{},
			local:   &LocalConfig{},
			wantErr: false,
		},
		{
			name:    "exact match returns no error",
			cfg:     &Config{Profiles: []string{"personal", "work"}},
			local:   &LocalConfig{Profiles: map[string]Profile{"personal": {}, "work": {}}},
			wantErr: false,
		},
		{
			name:       "declared profile missing in local",
			cfg:        &Config{Profiles: []string{"personal"}},
			local:      &LocalConfig{Profiles: map[string]Profile{}},
			wantErr:    true,
			sentinels:  []error{ErrMissingLocalProfile},
			substrings: []string{"personal", "missing in yhub.local.toml"},
		},
		{
			name:       "local profile not declared in main",
			cfg:        &Config{},
			local:      &LocalConfig{Profiles: map[string]Profile{"personal": {}}},
			wantErr:    true,
			sentinels:  []error{ErrUndeclaredLocalProfile},
			substrings: []string{"personal", "not declared in yhub.toml"},
		},
		{
			name:       "nil local with declared profiles reports missing",
			cfg:        &Config{Profiles: []string{"personal"}},
			local:      nil,
			wantErr:    true,
			sentinels:  []error{ErrMissingLocalProfile},
			substrings: []string{"personal"},
		},
		{
			name:    "nil local with no profiles is ok",
			cfg:     &Config{},
			local:   nil,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateProfiles(tt.cfg, tt.local)

			if !tt.wantErr {
				require.NoError(t, err)
				return
			}

			require.Error(t, err)
			for _, sentinel := range tt.sentinels {
				assert.ErrorIs(t, err, sentinel)
			}
			msg := err.Error()
			for _, sub := range tt.substrings {
				assert.True(t, strings.Contains(msg, sub),
					"expected error message to contain %q, got: %s", sub, msg)
			}
		})
	}
}
