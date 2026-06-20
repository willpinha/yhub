package config

import (
	"errors"
	"fmt"
	iofs "io/fs"

	"github.com/pelletier/go-toml/v2"
	"github.com/spf13/afero"
)

const FileName = "yhub.toml"

const LocalFileName = "yhub.local.toml"

var ErrNotFound = errors.New("config file not found")

var ErrInvalidTOML = errors.New("invalid TOML in config file")

type Profile struct {
	Name   string `toml:"name"`
	Email  string `toml:"email"`
	SSHKey string `toml:"ssh_key"`
}

type Platform struct {
	Host string `toml:"host"`
}

type Repository struct {
	Profile    string `toml:"profile"`
	Repository string `toml:"repository"`
	Name       string `toml:"name"`
	Alias      string `toml:"alias"`
	Platform   string `toml:"platform"`
	Protocol   string `toml:"protocol"`
}

type Config struct {
	RepositoriesDir string                  `toml:"repositories_dir"`
	DefaultPlatform string                  `toml:"default_platform"`
	DefaultProtocol string                  `toml:"default_protocol"`
	Profiles        []string                `toml:"profiles"`
	Platforms       map[string]Platform     `toml:"platforms"`
	Groups          map[string][]Repository `toml:"groups"`
}

type LocalConfig struct {
	Profiles map[string]Profile `toml:"profiles"`
}

func Load(fs afero.Fs, path string) (*Config, error) {
	data, err := afero.ReadFile(fs, path)
	if err != nil {
		if errors.Is(err, iofs.ErrNotExist) {
			return nil, fmt.Errorf("%w: %s", ErrNotFound, path)
		}
		return nil, fmt.Errorf("could not read config file: %w", err)
	}

	var cfg Config
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidTOML, err)
	}

	if cfg.RepositoriesDir == "" {
		cfg.RepositoriesDir = "repositories"
	}

	return &cfg, nil
}

func LoadLocal(fs afero.Fs, path string) (*LocalConfig, error) {
	data, err := afero.ReadFile(fs, path)
	if err != nil {
		if errors.Is(err, iofs.ErrNotExist) {
			return nil, fmt.Errorf("%w: %s", ErrNotFound, path)
		}
		return nil, fmt.Errorf("could not read config file: %w", err)
	}

	var cfg LocalConfig
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidTOML, err)
	}

	return &cfg, nil
}
