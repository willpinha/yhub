package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/willpinha/yhub/internal/config"
)

func TestRenderList_AlphabeticalGroupOrder(t *testing.T) {
	cfg := &config.Config{
		Groups: map[string][]config.Repository{
			"hello": {
				{Alias: "WRD", Name: "world", Repository: "willpinha/world", Profile: "personal"},
			},
			"foo": {
				{Alias: "BR", Name: "bar", Repository: "willpinha/bar", Profile: "work"},
			},
		},
	}

	var buf bytes.Buffer
	renderList(&buf, cfg)
	out := buf.String()

	fooIdx := strings.Index(out, "foo")
	helloIdx := strings.Index(out, "hello")

	assert.Less(t, fooIdx, helloIdx, "group 'foo' should appear before group 'hello'")
}

func TestRenderList_ReposKeepSliceOrder(t *testing.T) {
	cfg := &config.Config{
		Groups: map[string][]config.Repository{
			"foo": {
				{Alias: "BR", Name: "bar", Repository: "willpinha/bar", Profile: "work"},
				{Alias: "BZ", Name: "baz", Repository: "willpinha/baz", Profile: ""},
			},
		},
	}

	var buf bytes.Buffer
	renderList(&buf, cfg)
	out := buf.String()

	brIdx := strings.Index(out, "BR")
	bzIdx := strings.Index(out, "BZ")

	assert.Less(t, brIdx, bzIdx, "BR should appear before BZ (original slice order)")
}

func TestRenderList_EmptyProfileShowsDash(t *testing.T) {
	cfg := &config.Config{
		Groups: map[string][]config.Repository{
			"foo": {
				{Alias: "BZ", Name: "baz", Repository: "willpinha/baz", Profile: ""},
			},
		},
	}

	var buf bytes.Buffer
	renderList(&buf, cfg)
	out := buf.String()

	assert.Contains(t, out, "-")
}

func TestRenderList_NonEmptyProfileRendersVerbatim(t *testing.T) {
	cfg := &config.Config{
		Groups: map[string][]config.Repository{
			"foo": {
				{Alias: "BR", Name: "bar", Repository: "willpinha/bar", Profile: "work"},
			},
		},
	}

	var buf bytes.Buffer
	renderList(&buf, cfg)
	out := buf.String()

	assert.Contains(t, out, "work")
}

func TestRenderList_GroupNameAndRowValues(t *testing.T) {
	cfg := &config.Config{
		Groups: map[string][]config.Repository{
			"foo": {
				{Alias: "BR", Name: "bar", Repository: "willpinha/bar", Profile: "work"},
			},
		},
	}

	var buf bytes.Buffer
	renderList(&buf, cfg)
	out := buf.String()

	assert.Contains(t, out, "foo")
	assert.Contains(t, out, "BR")
	assert.Contains(t, out, "bar")
	assert.Contains(t, out, "willpinha/bar")
}

func TestRenderList_EmptyConfig(t *testing.T) {
	cfg := &config.Config{}

	var buf bytes.Buffer
	renderList(&buf, cfg)

	assert.Equal(t, "no repositories declared in yhub.toml\n", buf.String())
}
