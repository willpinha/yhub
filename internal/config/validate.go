package config

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
)

var (
	ErrEmptyField          = errors.New("empty required field")
	ErrUnknownProfile      = errors.New("unknown profile")
	ErrDuplicateAlias      = errors.New("duplicate alias")
	ErrDuplicateName       = errors.New("duplicate name")
	ErrBadRepositoryFormat = errors.New("bad repository format")
)

var repositoryRe = regexp.MustCompile(`^[^/\s]+/[^/\s]+$`)

type violation struct {
	sentinel error
	msg      string
}

func (v *violation) Error() string { return v.msg }
func (v *violation) Unwrap() error { return v.sentinel }

func newViolation(sentinel error, format string, args ...any) error {
	return &violation{
		sentinel: sentinel,
		msg:      fmt.Sprintf(format, args...),
	}
}

func sortedGroupNames(c *Config) []string {
	names := make([]string, 0, len(c.Groups))
	for g := range c.Groups {
		names = append(names, g)
	}
	sort.Strings(names)
	return names
}

type Validator func(c *Config) []error

func validateRequiredFields(c *Config) []error {
	var errs []error
	for _, group := range sortedGroupNames(c) {
		for _, repo := range c.Groups[group] {
			if repo.Repository == "" {
				errs = append(errs, newViolation(
					ErrEmptyField,
					`group %q repo %q: field "repository" is empty`,
					group, repo.Alias,
				))
			}
			if repo.Name == "" {
				errs = append(errs, newViolation(
					ErrEmptyField,
					`group %q repo %q: field "name" is empty`,
					group, repo.Alias,
				))
			}
			if repo.Alias == "" {
				errs = append(errs, newViolation(
					ErrEmptyField,
					`group %q: field "alias" is empty`,
					group,
				))
			}
		}
	}
	return errs
}

func validateProfiles(c *Config) []error {
	profileSet := make(map[string]bool, len(c.Profiles))
	for _, p := range c.Profiles {
		profileSet[p] = true
	}
	var errs []error
	for _, group := range sortedGroupNames(c) {
		for _, repo := range c.Groups[group] {
			if repo.Profile != "" && !profileSet[repo.Profile] {
				errs = append(errs, newViolation(
					ErrUnknownProfile,
					`group %q repo %q: profile %q is not declared in profiles`,
					group, repo.Alias, repo.Profile,
				))
			}
		}
	}
	return errs
}

func validateUniqueAliases(c *Config) []error {
	seen := make(map[string]string)
	var errs []error
	for _, group := range sortedGroupNames(c) {
		for _, repo := range c.Groups[group] {
			if repo.Alias == "" {
				continue
			}
			loc := fmt.Sprintf("group %q", group)
			if prev, ok := seen[repo.Alias]; ok {
				errs = append(errs, newViolation(
					ErrDuplicateAlias,
					`alias %q is used in both %s and %s`,
					repo.Alias, prev, loc,
				))
			} else {
				seen[repo.Alias] = loc
			}
		}
	}
	return errs
}

func validateUniqueNames(c *Config) []error {
	seen := make(map[string]string)
	var errs []error
	for _, group := range sortedGroupNames(c) {
		for _, repo := range c.Groups[group] {
			if repo.Name == "" {
				continue
			}
			loc := fmt.Sprintf("group %q", group)
			if prev, ok := seen[repo.Name]; ok {
				errs = append(errs, newViolation(
					ErrDuplicateName,
					`name %q is used in both %s and %s`,
					repo.Name, prev, loc,
				))
			} else {
				seen[repo.Name] = loc
			}
		}
	}
	return errs
}

func validateRepositoryFormat(c *Config) []error {
	var errs []error
	for _, group := range sortedGroupNames(c) {
		for _, repo := range c.Groups[group] {
			if repo.Repository != "" && !repositoryRe.MatchString(repo.Repository) {
				errs = append(errs, newViolation(
					ErrBadRepositoryFormat,
					`group %q repo %q: repository %q does not match "owner/repo" format`,
					group, repo.Alias, repo.Repository,
				))
			}
		}
	}
	return errs
}

func (c *Config) Validate() error {
	validators := []Validator{
		validateRequiredFields,
		validateProfiles,
		validateUniqueAliases,
		validateUniqueNames,
		validateRepositoryFormat,
	}
	var violations []error
	for _, v := range validators {
		violations = append(violations, v(c)...)
	}
	return errors.Join(violations...)
}
