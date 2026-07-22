package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os/exec"
	"path"
	"strings"

	"github.com/spf13/afero"
)

func (c *Config) Unclone(ctx context.Context, identifiers []string, force bool) error {
	processed := Set[string]{}
	failed := false

	for _, identifier := range identifiers {
		repo, err := c.FindRepository(identifier)
		if err != nil {
			slog.Warn("no repository found with name or alias", "identifier", identifier)
			continue
		}

		if processed.Has(repo.Path) {
			continue
		}
		processed.Add(repo.Path)

		cloned, err := afero.DirExists(c.fs, repo.Path)
		if err != nil {
			slog.Error("failed to check repository directory", "repository", repo.Repository, "error", err)
			failed = true
			continue
		}

		if !cloned {
			slog.Warn("repository is not cloned", "repository", repo.Repository, "path", repo.Path)
			continue
		}

		if !c.uncloneRepository(ctx, repo, force) {
			failed = true
		}
	}

	return uncloneError(failed)
}

func (c *Config) UncloneAll(ctx context.Context, force bool) error {
	anyCloned, failed := c.uncloneMatching(ctx, func(string) bool { return true }, force)

	if !anyCloned && !failed {
		slog.Warn("no repositories are cloned")
	}

	return uncloneError(failed)
}

func (c *Config) UncloneDir(ctx context.Context, dir string, force bool) error {
	dir = path.Clean(dir)

	matched := false
	for configDir := range c.Repositories {
		if insideDir(configDir, dir) {
			matched = true
			break
		}
	}

	if !matched {
		slog.Warn("no configured directory matches", "directory", dir)
		return nil
	}

	anyCloned, failed := c.uncloneMatching(ctx, func(configDir string) bool {
		return insideDir(configDir, dir)
	}, force)

	if !anyCloned && !failed {
		slog.Warn("no repositories are cloned in directory", "directory", dir)
	}

	return uncloneError(failed)
}

func (c *Config) uncloneMatching(ctx context.Context, match func(dir string) bool, force bool) (anyCloned, failed bool) {
	for dir, repo := range c.Repositories.All() {
		if !match(dir) {
			continue
		}

		result := c.newSearchResult(dir, repo)

		cloned, err := afero.DirExists(c.fs, result.Path)
		if err != nil {
			slog.Error("failed to check repository directory", "repository", result.Repository, "error", err)
			failed = true
			continue
		}

		if !cloned {
			continue
		}
		anyCloned = true

		if !c.uncloneRepository(ctx, result, force) {
			failed = true
		}
	}

	return anyCloned, failed
}

// Compares whole path segments, so "workshop" is not inside "work"
func insideDir(configDir, dir string) bool {
	return configDir == dir || strings.HasPrefix(configDir, dir+"/")
}

func uncloneError(failed bool) error {
	if failed {
		return errors.New("failed to unclone one or more repositories")
	}

	return nil
}

func (c *Config) uncloneRepository(ctx context.Context, repo SearchResult, force bool) bool {
	if !force {
		unsaved, err := unsavedWork(ctx, c.fs, repo.Path)
		if err != nil {
			slog.Error("failed to check for unsaved work", "repository", repo.Repository, "path", repo.Path, "error", err)
			return false
		}

		if unsaved != "" {
			slog.Warn("repository has unsaved work", "repository", repo.Repository, "path", repo.Path, "reason", unsaved)
			return true
		}
	}

	if err := c.fs.RemoveAll(repo.Path); err != nil {
		slog.Error("failed to remove repository directory", "repository", repo.Repository, "path", repo.Path, "error", err)
		return false
	}

	slog.Info("repository removed", "repository", repo.Repository, "path", repo.Path)

	return true
}

// The .git check must happen before running git, which would otherwise resolve
// a directory that is not a repository to a parent one (the hub itself)
func unsavedWork(ctx context.Context, fs afero.Fs, dir string) (string, error) {
	hasGitDir, err := afero.Exists(fs, path.Join(dir, ".git"))
	if err != nil {
		return "", err
	}

	if !hasGitDir {
		return "", errors.New("directory is not a git repository")
	}

	status, err := gitOutput(ctx, dir, "status", "--porcelain")
	if err != nil {
		return "", err
	}

	if status != "" {
		return "uncommitted changes", nil
	}

	unpushed, err := gitOutput(ctx, dir, "log", "--branches", "--not", "--remotes", "--oneline")
	if err != nil {
		return "", err
	}

	if unpushed != "" {
		return "unpushed commits", nil
	}

	return "", nil
}

func gitOutput(ctx context.Context, dir string, args ...string) (string, error) {
	var stdout, stderr bytes.Buffer

	command := exec.CommandContext(ctx, "git", args...)
	command.Dir = dir
	command.Stdout = &stdout
	command.Stderr = &stderr

	if err := command.Run(); err != nil {
		message := strings.TrimSpace(stderr.String())
		if message == "" {
			message = err.Error()
		}

		return "", fmt.Errorf("git %s failed: %s", strings.Join(args, " "), message)
	}

	return strings.TrimSpace(stdout.String()), nil
}
