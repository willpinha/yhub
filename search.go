package main

import (
	"path"
	"strings"
	"unicode"
	"unicode/utf8"
)

type SearchResult struct {
	Repository string   `json:"repository"`
	Name       string   `json:"name"`
	Aliases    []string `json:"aliases"`
	Platform   string   `json:"platform"`
	Profile    string   `json:"profile"`
	Directory  string   `json:"directory"`
	Path       string   `json:"path"`
}

func (c *Config) Search(text string) []SearchResult {
	// Non-nil so an empty result serializes as [] instead of null
	results := []SearchResult{}

	for dir, repo := range c.Repositories.All() {
		if mentionsRepository(text, repo) {
			results = append(results, c.newSearchResult(dir, repo))
		}
	}

	return results
}

func (c *Config) newSearchResult(dir string, repo Repository) SearchResult {
	platform := repo.Platform
	if platform == "" {
		platform = c.DefaultPlatform
	}

	profile := repo.Profile
	if profile == "" {
		profile = c.DefaultProfile
	}

	aliases := repo.Aliases
	if aliases == nil {
		aliases = []string{}
	}

	return SearchResult{
		Repository: repo.Repository,
		Name:       repo.Name,
		Aliases:    aliases,
		Platform:   platform,
		Profile:    profile,
		Directory:  dir,
		Path:       path.Join(dir, repo.Name),
	}
}

func mentionsRepository(text string, repo Repository) bool {
	for _, id := range append([]string{repo.Name}, repo.Aliases...) {
		if mentions(text, id) {
			return true
		}
	}

	return false
}

// A mention is a case-insensitive occurrence of the identifier that is not
// adjacent to word characters (letters, digits, '_' or '-'), so "hello-world"
// is not mentioned in "hey-hello-world" or "hello-worlds"
func mentions(text, identifier string) bool {
	text = strings.ToLower(text)
	identifier = strings.ToLower(identifier)

	for start := 0; ; start++ {
		i := strings.Index(text[start:], identifier)
		if i < 0 {
			return false
		}

		start += i
		end := start + len(identifier)

		if !wordRuneBefore(text, start) && !wordRuneAfter(text, end) {
			return true
		}
	}
}

func wordRuneBefore(text string, i int) bool {
	r, size := utf8.DecodeLastRuneInString(text[:i])

	return size > 0 && isWordRune(r)
}

func wordRuneAfter(text string, i int) bool {
	r, size := utf8.DecodeRuneInString(text[i:])

	return size > 0 && isWordRune(r)
}

func isWordRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-'
}
