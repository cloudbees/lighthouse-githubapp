package util_test

import (
	"testing"

	"github.com/cloudbees/lighthouse-githubapp/pkg/util"
	"github.com/stretchr/testify/assert"
)

func TestUrlJoin(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "http://foo.bar/whatnot/thingy", util.UrlJoin("http://foo.bar", "whatnot", "thingy"))
	assert.Equal(t, "http://foo.bar/whatnot/thingy/", util.UrlJoin("http://foo.bar/", "/whatnot/", "/thingy/"))
}
