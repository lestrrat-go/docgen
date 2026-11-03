// Package basic is the root package of the basic fixture.
//
// # Usage
//
// Call [Do] with these inputs:
//
//   - a reader
//   - a circle
//
// Sample code:
//
//	err := basic.Do(r, c, t, s)
//
// See https://example.com/docs for details, or the [Go site].
//
// The doc links are [Version], [shape.Circle], [io.Reader],
// [example.com/basic/internal/secret.Secret] and [nothing].
//
// [Go site]: https://go.dev
package basic

import (
	"errors"
	"io"

	"example.com/basic/internal/secret"
	"example.com/basic/shape"
	"example.com/thirdparty"
)

// Version is the fixture version.
const Version = "1.0.0"

// Config holds settings.
type Config struct{}

// ErrNope is returned when nothing works.
var ErrNope = errors.New("nope")

// Do does things with its inputs.
func Do(r io.Reader, c *shape.Circle, t thirdparty.Thing, s secret.Secret) error {
	return ErrNope
}

// Old does nothing.
//
// Deprecated: use Do.
func Old() {}
