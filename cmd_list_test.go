package main

import (
	"context"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func runListCommand(t *testing.T, fs afero.Fs, args ...string) error {
	t.Helper()

	return ListCommand(fs).Run(context.Background(), append([]string{"list"}, args...))
}

func TestListCommandRejectsArguments(t *testing.T) {
	fs := writeConfig(t, baseConfig())

	err := runListCommand(t, fs, "extra")
	assert.ErrorContains(t, err, "expected no arguments")
}

func TestListCommandSucceeds(t *testing.T) {
	fs := writeConfig(t, baseConfig())

	assert.NoError(t, runListCommand(t, fs))
}
