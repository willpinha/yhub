package git

import "context"

type Git interface {
	// Clone clones url into dest. env contains extra KEY=VALUE entries
	// (e.g. GIT_SSH_COMMAND=...) appended on top of the inherited environment
	Clone(ctx context.Context, url, dest string, env []string) error

	// SetConfig runs "git config key value" inside repoPath
	SetConfig(ctx context.Context, repoPath, key, value string) error

	// HasUncommittedChanges reports whether repoPath has uncommitted changes
	// (staged or unstaged). It returns (true, nil) when the working tree is dirty
	HasUncommittedChanges(ctx context.Context, repoPath string) (bool, error)

	// HasUnpushedCommits reports whether repoPath has local commits not present
	// on any remote. Returns (true, nil) conservatively when no remote is configured
	HasUnpushedCommits(ctx context.Context, repoPath string) (bool, error)
}
