package main

import (
	"testing"

	"github.com/blendlabs/go-assert"
)

func TestReadMap(t *testing.T) {
	assert := assert.New(t)

	things := map[string]interface{}{
		"foo": map[string]interface{}{
			"bar": "baz",
		},
	}

	actual := readMap(things, "foo", "bar")
	assert.Equal("baz", actual)
}
