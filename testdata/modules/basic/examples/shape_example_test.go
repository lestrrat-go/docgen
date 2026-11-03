package examples_test

import (
	"fmt"

	"example.com/basic/shape"
)

func Example_shape_circle() {
	fmt.Println(shape.NewCircle(2).R)
	// Output: 2
}
