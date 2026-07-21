package main

import (
	"github.com/spf13/afero"
)

func (c *Config) ListCloned() ([]SearchResult, error) {
	// Non-nil so an empty result serializes as [] instead of null
	results := []SearchResult{}

	for dir, repo := range c.Repositories.All() {
		result := c.newSearchResult(dir, repo)

		cloned, err := afero.DirExists(c.fs, result.Path)
		if err != nil {
			return nil, err
		}

		if cloned {
			results = append(results, result)
		}
	}

	return results, nil
}
