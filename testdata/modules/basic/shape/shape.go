// Package shape defines shapes.
//
// A Circle wraps an io.Reader, never a bytes.Buffer. The root type basic.Config
// and the constant basic.Version live elsewhere. Markdown-looking text: *bold*,
// _x_, <nil>, &amp;, ~/path, a | b, $HOME and the numbered line
// 1999. was a good year.
package shape

import "io"

// Kind enumerates kinds.
type Kind int

// Known kinds.
const (
	KindA Kind = iota
	KindB
)

// DefaultKind is the default kind.
var DefaultKind Kind = KindA

// Sizes.
const (
	Small = 1
	Large = 2
)

// Arrow points left.
const Arrow = "<-"

// Shape is a shape.
type Shape interface {
	// Area returns the area.
	Area() float64
	// Name returns the name.
	Name() string
	unexported()
}

// Base is a base type.
type Base struct{}

type inner struct{}

// Hello greets.
func (i *inner) Hello() {}

// Circle is a circle, not a square.
type Circle struct {
	// R is the radius.
	R float64 // radius
	io.Reader
	Base
	inner
	hidden int
	// ByKind groups values by kind.
	ByKind map[string][]Kind
}

// NewCircle returns a circle.
func NewCircle(r float64) *Circle { return &Circle{R: r} }

// Area returns the area of c.
func (c Circle) Area() float64 { return c.R * c.R * 3 }

// Scale multiplies the radius.
func (c *Circle) Scale(f float64) { c.R *= f }

// List is a list of shapes.
type List[T Shape] struct {
	items []T
}

// Push appends a shape.
func (l *List[T]) Push(v T) { l.items = append(l.items, v) }

// Map applies f to each element of in.
func Map[T, U any](in []T, f func(T) U) []U {
	out := make([]U, 0, len(in))
	for _, v := range in {
		out = append(out, f(v))
	}
	return out
}

// Alias is another name for Circle.
type Alias = Circle

// OldShape is old.
//
// Deprecated: use Circle.
type OldShape struct{}
