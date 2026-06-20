package gittest

import (
	"context"

	"github.com/willpinha/yhub/internal/git"
)

var _ git.Git = (*FakeGit)(nil)

type CloneCall struct {
	URL, Dest string
	Env       []string
}

type SetConfigCall struct {
	RepoPath, Key, Value string
}

type FakeGit struct {
	Clones                    []CloneCall
	SetConfigs                []SetConfigCall
	HasUncommittedChangesRepo []string
	HasUnpushedCommitsRepo    []string

	CloneFunc                 func(ctx context.Context, url, dest string, env []string) error
	SetConfigFunc             func(ctx context.Context, repoPath, key, value string) error
	HasUncommittedChangesFunc func(ctx context.Context, repoPath string) (bool, error)
	HasUnpushedCommitsFunc    func(ctx context.Context, repoPath string) (bool, error)
}

func (f *FakeGit) Clone(ctx context.Context, url, dest string, env []string) error {
	f.Clones = append(f.Clones, CloneCall{URL: url, Dest: dest, Env: env})
	if f.CloneFunc != nil {
		return f.CloneFunc(ctx, url, dest, env)
	}
	return nil
}

func (f *FakeGit) SetConfig(ctx context.Context, repoPath, key, value string) error {
	f.SetConfigs = append(f.SetConfigs, SetConfigCall{RepoPath: repoPath, Key: key, Value: value})
	if f.SetConfigFunc != nil {
		return f.SetConfigFunc(ctx, repoPath, key, value)
	}
	return nil
}

func (f *FakeGit) HasUncommittedChanges(ctx context.Context, repoPath string) (bool, error) {
	f.HasUncommittedChangesRepo = append(f.HasUncommittedChangesRepo, repoPath)
	if f.HasUncommittedChangesFunc != nil {
		return f.HasUncommittedChangesFunc(ctx, repoPath)
	}
	return false, nil
}

func (f *FakeGit) HasUnpushedCommits(ctx context.Context, repoPath string) (bool, error) {
	f.HasUnpushedCommitsRepo = append(f.HasUnpushedCommitsRepo, repoPath)
	if f.HasUnpushedCommitsFunc != nil {
		return f.HasUnpushedCommitsFunc(ctx, repoPath)
	}
	return false, nil
}
