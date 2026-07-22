package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/spf13/afero"
)

func (c *Config) Clone(ctx context.Context, identifiers []string) error {
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

		if cloned {
			slog.Warn("repository is already cloned", "repository", repo.Repository, "path", repo.Path)
			continue
		}

		if !c.cloneRepository(ctx, repo) {
			failed = true
		}
	}

	return cloneError(failed)
}

func (c *Config) CloneAll(ctx context.Context) error {
	anyMissing, failed := c.cloneMatching(ctx, func(string) bool { return true })

	if !anyMissing && !failed {
		slog.Warn("all repositories are already cloned")
	}

	return cloneError(failed)
}

func (c *Config) CloneDir(ctx context.Context, dir string) error {
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

	anyMissing, failed := c.cloneMatching(ctx, func(configDir string) bool {
		return insideDir(configDir, dir)
	})

	if !anyMissing && !failed {
		slog.Warn("all repositories are already cloned in directory", "directory", dir)
	}

	return cloneError(failed)
}

func (c *Config) cloneMatching(ctx context.Context, match func(dir string) bool) (anyMissing, failed bool) {
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

		if cloned {
			continue
		}
		anyMissing = true

		if !c.cloneRepository(ctx, result) {
			failed = true
		}
	}

	return anyMissing, failed
}

func cloneError(failed bool) error {
	if failed {
		return errors.New("failed to clone one or more repositories")
	}

	return nil
}

// A variable so tests can replace the network-dependent git execution. The
// command inherits the terminal so the user can see the clone progress and
// answer known_hosts or key passphrase prompts
var gitClone = func(ctx context.Context, args []string) error {
	command := exec.CommandContext(ctx, "git", args...)
	command.Stdin = os.Stdin
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr

	return command.Run()
}

func (c *Config) cloneRepository(ctx context.Context, repo SearchResult) bool {
	profile := c.Profiles[repo.Profile]

	sshKey, err := expandHomePath(profile.SSHKey)
	if err != nil {
		slog.Error("failed to resolve SSH key path", "repository", repo.Repository, "ssh_key", profile.SSHKey, "error", err)
		return false
	}

	exists, err := afero.Exists(c.fs, sshKey)
	if err != nil {
		slog.Error("failed to check SSH key", "repository", repo.Repository, "ssh_key", profile.SSHKey, "path", sshKey, "error", err)
		return false
	}

	if !exists {
		slog.Error("SSH key does not exist", "repository", repo.Repository, "ssh_key", profile.SSHKey, "path", sshKey)
		return false
	}

	slog.Info("cloning repository", "repository", repo.Repository, "path", repo.Path)

	url := cloneURL(c.Platforms[repo.Platform].Host, repo.Repository)

	if err := gitClone(ctx, cloneArgs(url, repo.Path, profile, sshKey)); err != nil {
		slog.Error("failed to clone repository", "repository", repo.Repository, "path", repo.Path, "error", err)
		return false
	}

	slog.Info("repository cloned", "repository", repo.Repository, "path", repo.Path)

	return true
}

// The --config values are used during the clone and persisted in the local git
// config, so later git commands in the repository need no extra authentication
// or identity setup
func cloneArgs(url, destination string, profile Profile, sshKey string) []string {
	return []string{
		"clone",
		"--config", "core.sshCommand=" + sshCommand(sshKey),
		"--config", "user.name=" + profile.UserName,
		"--config", "user.email=" + profile.UserEmail,
		url, destination,
	}
}

func cloneURL(host, repository string) string {
	return fmt.Sprintf("git@%s:%s.git", host, repository)
}

// Single quotes so the shell that runs core.sshCommand never expands anything
// in the key path
func sshCommand(sshKey string) string {
	quoted := "'" + strings.ReplaceAll(sshKey, "'", `'\''`) + "'"

	return fmt.Sprintf("ssh -i %s -o IdentitiesOnly=yes", quoted)
}

func expandHomePath(p string) (string, error) {
	if p != "~" && !strings.HasPrefix(p, "~/") {
		return p, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return path.Join(home, strings.TrimPrefix(p, "~")), nil
}
