package gittest_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/willpinha/yhub/internal/git/gittest"
)

func TestFakeGit_RecordsCloneCall(t *testing.T) {
	f := &gittest.FakeGit{}
	ctx := context.Background()

	err := f.Clone(ctx, "https://example.com/repo.git", "/tmp/repo", nil)

	require.NoError(t, err)
	require.Len(t, f.Clones, 1)
	assert.Equal(t, "https://example.com/repo.git", f.Clones[0].URL)
	assert.Equal(t, "/tmp/repo", f.Clones[0].Dest)
}

func TestFakeGit_CloneFuncReturnsError(t *testing.T) {
	cloneErr := errors.New("network failure")
	f := &gittest.FakeGit{
		CloneFunc: func(_ context.Context, _, _ string, _ []string) error {
			return cloneErr
		},
	}

	err := f.Clone(context.Background(), "https://example.com/repo.git", "/tmp/repo", nil)

	require.ErrorIs(t, err, cloneErr)
	require.Len(t, f.Clones, 1, "call must be recorded even when func returns error")
}

func TestFakeGit_RecordsSetConfigCall(t *testing.T) {
	f := &gittest.FakeGit{}

	err := f.SetConfig(context.Background(), "/repo", "user.name", "Alice")

	require.NoError(t, err)
	require.Len(t, f.SetConfigs, 1)
	assert.Equal(t, gittest.SetConfigCall{RepoPath: "/repo", Key: "user.name", Value: "Alice"}, f.SetConfigs[0])
}

func TestFakeGit_DefaultBooleanMethodsReturnFalse(t *testing.T) {
	f := &gittest.FakeGit{}
	ctx := context.Background()

	dirty, err := f.HasUncommittedChanges(ctx, "/repo")
	require.NoError(t, err)
	assert.False(t, dirty)

	unpushed, err := f.HasUnpushedCommits(ctx, "/repo")
	require.NoError(t, err)
	assert.False(t, unpushed)

	assert.Equal(t, []string{"/repo"}, f.HasUncommittedChangesRepo)
	assert.Equal(t, []string{"/repo"}, f.HasUnpushedCommitsRepo)
}

func TestFakeGit_HasUncommittedChangesFuncOverride(t *testing.T) {
	f := &gittest.FakeGit{
		HasUncommittedChangesFunc: func(_ context.Context, _ string) (bool, error) {
			return true, nil
		},
	}

	dirty, err := f.HasUncommittedChanges(context.Background(), "/repo")

	require.NoError(t, err)
	assert.True(t, dirty)
}
