package config

import (
	"errors"
	"fmt"
)

const defaultBuiltinPlatform = "github"
const defaultBuiltinProtocol = "https"

var ErrUnresolvablePlatform = errors.New("unresolvable platform")
var ErrUnresolvableRepository = errors.New("unresolvable repository")
var ErrUnresolvableProtocol = errors.New("unresolvable protocol")

func (c *Config) ResolvePlatform(repo Repository) string {
	if repo.Platform != "" {
		return repo.Platform
	}
	if c.DefaultPlatform != "" {
		return c.DefaultPlatform
	}
	return defaultBuiltinPlatform
}

func (c *Config) ResolveProtocol(repo Repository) string {
	if repo.Protocol != "" {
		return repo.Protocol
	}
	if c.DefaultProtocol != "" {
		return c.DefaultProtocol
	}
	return defaultBuiltinProtocol
}

func (c *Config) CloneURL(repo Repository) (string, error) {
	platform := c.ResolvePlatform(repo)
	protocol := c.ResolveProtocol(repo)

	host, ok := builtinPlatforms[platform]
	if !ok {
		if p, exists := c.Platforms[platform]; exists {
			host = p.Host
		}
	}
	if host == "" {
		return "", fmt.Errorf("%w: cannot resolve host for platform %q", ErrUnresolvablePlatform, platform)
	}

	if !repositoryRe.MatchString(repo.Repository) {
		return "", fmt.Errorf("%w: cannot build clone URL from repository %q", ErrUnresolvableRepository, repo.Repository)
	}

	switch protocol {
	case "https":
		return fmt.Sprintf("https://%s/%s.git", host, repo.Repository), nil
	case "ssh":
		return fmt.Sprintf("git@%s:%s.git", host, repo.Repository), nil
	default:
		return "", fmt.Errorf("%w: cannot build clone URL for protocol %q", ErrUnresolvableProtocol, protocol)
	}
}
