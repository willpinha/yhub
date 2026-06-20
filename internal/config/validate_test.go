package config

import (
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
