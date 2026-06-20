package config

import (
	"errors"
	"fmt"
	iofs "io/fs"

	"github.com/pelletier/go-toml/v2"
	"github.com/spf13/afero"
)

var ErrNotFound = errors.New("config file not found")

var ErrInvalidTOML = errors.New("invalid TOML in config file")

type Repository struct {
	Profile    string `toml:"profile"`
	Repository string `toml:"repository"`
	Name       string `toml:"name"`
	Alias      string `toml:"alias"`
}

type Config struct {
	RepositoriesDir string                  `toml:"repositories_dir"`
	Profiles        []string                `toml:"profiles"`
	Groups          map[string][]Repository `toml:"groups"`
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
