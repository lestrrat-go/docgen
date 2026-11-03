package shape_test

import (
	"fmt"

	"example.com/basic/shape"
)

func ExampleNewCircle() {
	c := shape.NewCircle(1)
	fmt.Println(c.R)
	// Output: 1
}
