// Command docgen generates Markdown API documentation for a Go module.
package main

import (
	"context"
	"os"

	"github.com/lestrrat-go/docgen/internal/cli"
)

func main() { os.Exit(cli.Run(context.Background(), os.Args[1:], os.Stdout, os.Stderr)) }
