package cli

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/willpinha/yhub/internal/config"
)

func TestCollectViolations(t *testing.T) {
	tests := []struct {
		name     string
		cfg      *config.Config
		local    *config.LocalConfig
		localErr error
		wantLen  int
		contains []string
	}{
		{
			name:     "happy path no violations",
			cfg:      &config.Config{},
			local:    &config.LocalConfig{},
			localErr: nil,
			wantLen:  0,
		},
		{
			name:     "local missing and profiles declared",
			cfg:      &config.Config{Profiles: []string{"personal", "work"}},
			local:    nil,
			localErr: fmt.Errorf("%w: yhub.local.toml", config.ErrNotFound),
			wantLen:  1,
			contains: []string{"yhub.local.toml not found", "2 profile(s)"},
		},
		{
			name:     "local missing and no profiles declared",
			cfg:      &config.Config{},
			local:    nil,
			localErr: fmt.Errorf("%w: yhub.local.toml", config.ErrNotFound),
			wantLen:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			violations := collectViolations(tt.cfg, tt.local, tt.localErr)
			assert.Len(t, violations, tt.wantLen)
			if tt.wantLen > 0 {
				msg := errors.Join(violations...).Error()
				for _, sub := range tt.contains {
					assert.Contains(t, msg, sub)
				}
			}
		})
	}
}
