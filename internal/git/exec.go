package git

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type execGit struct{}

func New() Git {
	return &execGit{}
}

func (g *execGit) run(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return string(out), nil
}

func (g *execGit) Clone(ctx context.Context, url, dest string, env []string) error {
	cmd := exec.CommandContext(ctx, "git", "clone", url, dest)
	cmd.Env = append(os.Environ(), env...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git clone %s: %w: %s", url, err, strings.TrimSpace(string(out)))
	}
	return nil
}

func (g *execGit) SetConfig(ctx context.Context, repoPath, key, value string) error {
	_, err := g.run(ctx, "-C", repoPath, "config", key, value)
	return err
}

func (g *execGit) HasUncommittedChanges(ctx context.Context, repoPath string) (bool, error) {
	out, err := g.run(ctx, "-C", repoPath, "status", "--porcelain")
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(out) != "", nil
}

func (g *execGit) HasUnpushedCommits(ctx context.Context, repoPath string) (bool, error) {
	out, err := g.run(ctx, "-C", repoPath, "log", "--branches", "--not", "--remotes", "--format=%H")
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(out) != "", nil
}
