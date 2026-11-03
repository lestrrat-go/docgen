// Package cli implements the docgen command line.
package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"runtime/debug"
	"slices"
	"strings"

	"github.com/lestrrat-go/docgen/internal/link"
	"github.com/lestrrat-go/docgen/internal/load"
	"github.com/lestrrat-go/docgen/internal/output"
	"github.com/lestrrat-go/docgen/internal/render"
	"github.com/lestrrat-go/docgen/internal/source"
)

// Exit codes returned by Run.
const (
	ExitOK    = 0 // success; with --check, no differences
	ExitDiff  = 1 // --check found differences
	ExitError = 2 // usage error, load or type error, collision, I/O error, refused overwrite
)

// options holds the parsed flags.
type options struct {
	moduleDir     string
	out           string
	check         bool
	goos          string
	goarch        string
	tags          string
	sourceRef     string
	sourceBaseURL string
	version       bool
}

// Run executes docgen with args (without the program name). It writes the --check
// report and --version output to stdout, and all diagnostics to stderr.
func Run(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	var opts options
	fs := flag.NewFlagSet("docgen", flag.ContinueOnError)
	// The flag package prints its own error and usage text; docgen prints diagnostics itself.
	fs.SetOutput(io.Discard)
	fs.StringVar(&opts.moduleDir, "module-dir", ".", "module root; must contain go.mod")
	fs.StringVar(&opts.out, "out", "docs", "output directory; a relative path is joined with --module-dir")
	fs.BoolVar(&opts.check, "check", false, "write nothing; print differing paths; exit 1 if any")
	fs.StringVar(&opts.goos, "goos", "linux", "GOOS for the build context")
	fs.StringVar(&opts.goarch, "goarch", "amd64", "GOARCH for the build context")
	fs.StringVar(&opts.tags, "tags", "", "comma-separated build tags")
	fs.StringVar(&opts.sourceRef, "source-ref", "", "enable GitHub source links at this ref")
	fs.StringVar(&opts.sourceBaseURL, "source-base-url", "", "enable source links under this base URL")
	fs.BoolVar(&opts.version, "version", false, "print the version and exit")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fs.SetOutput(stderr)
			_, _ = fmt.Fprintln(stderr, "Usage: docgen [flags]")
			fs.PrintDefaults()
			return ExitOK
		}
		return fail(stderr, err)
	}
	if fs.NArg() > 0 {
		return fail(stderr, fmt.Errorf("unexpected argument %q", fs.Arg(0)))
	}

	if opts.version {
		_, _ = fmt.Fprintf(stdout, "docgen %s\n", version())
		return ExitOK
	}

	if err := opts.validate(); err != nil {
		return fail(stderr, err)
	}

	moduleDir, err := filepath.Abs(opts.moduleDir)
	if err != nil {
		return fail(stderr, err)
	}
	outDir := opts.out
	if filepath.IsAbs(outDir) {
		outDir = filepath.Clean(outDir)
	} else {
		outDir = filepath.Join(moduleDir, outDir)
	}
	if containsDir(outDir, moduleDir) {
		return fail(stderr, fmt.Errorf("--out %s must not contain the module directory", outDir))
	}

	module, err := load.Load(ctx, load.Config{
		Dir:    moduleDir,
		GOOS:   opts.goos,
		GOARCH: opts.goarch,
		Tags:   splitTags(opts.tags),
	})
	if err != nil {
		return fail(stderr, err)
	}
	resolver, err := link.New(module)
	if err != nil {
		return fail(stderr, err)
	}

	// src stays a nil interface unless a linker is assigned; never assign a nil *source.Linker to it.
	var src render.SourceLinker
	if opts.sourceRef != "" {
		l, err := source.NewGitHub(moduleDir, module.Path, opts.sourceRef)
		if err != nil {
			return fail(stderr, err)
		}
		src = l
	}
	if opts.sourceBaseURL != "" {
		src = source.NewBase(opts.sourceBaseURL)
	}

	files, err := renderAll(module, resolver, src)
	if err != nil {
		return fail(stderr, err)
	}
	plan, err := output.Diff(outDir, files)
	if err != nil {
		return fail(stderr, err)
	}

	if opts.check {
		paths := plan.Paths()
		for _, p := range paths {
			_, _ = fmt.Fprintln(stdout, filepath.ToSlash(filepath.Join(opts.out, p)))
		}
		if len(paths) > 0 {
			return ExitDiff
		}
		return ExitOK
	}
	if err := output.Apply(outDir, files, plan); err != nil {
		return fail(stderr, err)
	}
	return ExitOK
}

func (o *options) validate() error {
	if o.goos == "" {
		return errors.New("--goos must not be empty")
	}
	if o.goarch == "" {
		return errors.New("--goarch must not be empty")
	}
	if o.sourceRef != "" && o.sourceBaseURL != "" {
		return errors.New("--source-ref and --source-base-url are mutually exclusive")
	}
	return nil
}

// fail prints err to stderr with the diagnostic prefix on every line and returns ExitError.
func fail(stderr io.Writer, err error) int {
	for line := range strings.SplitSeq(err.Error(), "\n") {
		_, _ = fmt.Fprintf(stderr, "docgen: %s\n", line)
	}
	return ExitError
}

func version() string {
	info, ok := debug.ReadBuildInfo()
	if !ok || info.Main.Version == "" {
		return "(devel)"
	}
	return info.Main.Version
}

// containsDir reports whether outDir equals or contains moduleDir.
func containsDir(outDir, moduleDir string) bool {
	rel, err := filepath.Rel(outDir, moduleDir)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func splitTags(s string) []string {
	var tags []string
	for t := range strings.SplitSeq(s, ",") {
		if t = strings.TrimSpace(t); t != "" {
			tags = append(tags, t)
		}
	}
	return tags
}

func renderAll(m *load.Module, r *link.Resolver, src render.SourceLinker) ([]output.File, error) {
	files := make([]output.File, 0, len(m.Packages)+1)
	for _, p := range m.Packages {
		content, err := render.Package(m, r, p, src)
		if err != nil {
			return nil, err
		}
		path, _ := r.File(p.ImportPath)
		files = append(files, output.File{Path: path, Content: content})
	}
	files = append(files, output.File{Path: link.IndexFile, Content: render.Index(m, r)})
	slices.SortFunc(files, func(a, b output.File) int { return strings.Compare(a.Path, b.Path) })
	return files, nil
}
