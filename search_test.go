package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func searchConfig(t *testing.T) *Config {
	t.Helper()

	config, err := NewConfig(writeConfig(t, baseConfig()))
	require.NoError(t, err)

	return config
}

func TestSearch(t *testing.T) {
	config := searchConfig(t)

	tests := []struct {
		name string
		text string
		want []string
	}{
		{
			name: "empty text",
			text: "",
			want: nil,
		},
		{
			name: "no mentions",
			text: "nothing to see here",
			want: nil,
		},
		{
			name: "mention by name",
			text: "clone the task-pool repository",
			want: []string{"task-pool"},
		},
		{
			name: "mention by alias",
			text: "update TP before deploying",
			want: []string{"task-pool"},
		},
		{
			name: "different case name is not mentioned",
			text: "Clone Task-Pool for me",
			want: nil,
		},
		{
			name: "different case alias is not mentioned",
			text: "update tp before deploying",
			want: nil,
		},
		{
			name: "identifier alone",
			text: "task-pool",
			want: []string{"task-pool"},
		},
		{
			name: "hyphenated identifiers are whole words",
			text: "use active-task-pool here",
			want: []string{"active-task-pool"},
		},
		{
			name: "no mention inside a longer word",
			text: "task-pooling is fun",
			want: nil,
		},
		{
			name: "no alias mention inside a longer alias",
			text: "run ATP now",
			want: []string{"active-task-pool"},
		},
		{
			name: "punctuation is a boundary",
			text: "(task-pool), right?",
			want: []string{"task-pool"},
		},
		{
			name: "end of sentence is a boundary",
			text: "check task-pool.",
			want: []string{"task-pool"},
		},
		{
			name: "slash is a boundary",
			text: "see task-pool/src for details",
			want: []string{"task-pool"},
		},
		{
			name: "underscore is not a boundary",
			text: "see task-pool_backup",
			want: nil,
		},
		{
			name: "digit is not a boundary",
			text: "see task-pool2",
			want: nil,
		},
		{
			name: "accented letter is not a boundary",
			text: "see task-poolé",
			want: nil,
		},
		{
			name: "name and alias mentions are deduplicated",
			text: "task-pool (aka TP)",
			want: []string{"task-pool"},
		},
		{
			name: "repeated mentions are deduplicated",
			text: "TP here, TP there",
			want: []string{"task-pool"},
		},
		{
			name: "boundary retry after an embedded occurrence",
			text: "task-pooling and task-pool",
			want: []string{"task-pool"},
		},
		{
			name: "multiple repositories in config order",
			text: "TP, plan-assignment and ATP",
			want: []string{"plan-assignment", "active-task-pool", "task-pool"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var names []string
			for _, result := range config.Search(tt.text) {
				names = append(names, result.Name)
			}

			assert.Equal(t, tt.want, names)
		})
	}
}

func TestSearchResolvesDefaultsAndPaths(t *testing.T) {
	results := searchConfig(t).Search("plan-assignment and ATP")
	require.Len(t, results, 2)

	plan := results[0]
	assert.Equal(t, "example-user/plan-assignment", plan.Repository)
	assert.Equal(t, "plan-assignment", plan.Name)
	assert.Equal(t, []string{}, plan.Aliases)
	assert.Equal(t, "github", plan.Platform)
	assert.Equal(t, "personal", plan.Profile)
	assert.Equal(t, "personal/static-routing", plan.Directory)
	assert.Equal(t, "personal/static-routing/plan-assignment", plan.Path)

	atp := results[1]
	assert.Equal(t, "company/active-task-pool", atp.Repository)
	assert.Equal(t, []string{"ATP"}, atp.Aliases)
	assert.Equal(t, "company", atp.Platform)
	assert.Equal(t, "work", atp.Profile)
	assert.Equal(t, "work/dynamic-routing", atp.Directory)
	assert.Equal(t, "work/dynamic-routing/active-task-pool", atp.Path)
}

func TestFindRepository(t *testing.T) {
	config := searchConfig(t)

	byName, err := config.FindRepository("task-pool")
	require.NoError(t, err)
	assert.Equal(t, "company/task-pool", byName.Repository)
	assert.Equal(t, "work/dynamic-routing/task-pool", byName.Path)

	byAlias, err := config.FindRepository("ATP")
	require.NoError(t, err)
	assert.Equal(t, "company/active-task-pool", byAlias.Repository)

	_, err = config.FindRepository("unknown")
	assert.ErrorContains(t, err, "no repository found with name or alias 'unknown'")
}

func TestSearchWithoutMatchesReturnsEmptySlice(t *testing.T) {
	results := searchConfig(t).Search("nothing relevant")

	assert.NotNil(t, results)
	assert.Empty(t, results)
}
