// Deprecated: use package shape.
package three

import "example.com/basic/shape"

// Area returns the area of c.
func Area(c *shape.Circle) float64 { return c.Area() }
