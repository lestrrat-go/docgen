// Package load loads a Go module with full type information and builds go/doc packages for it.
package load

import (
	"context"
	"errors"
	"fmt"
	"go/ast"
	"go/doc"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"golang.org/x/mod/modfile"
	"golang.org/x/tools/go/packages"
)

// Config selects the module and build context. Dir MUST be an absolute path.
type Config struct {
	Dir    string
	GOOS   string
	GOARCH string
	Tags   []string
}

// Module is a loaded module.
type Module struct {
	Path     string              // module path from go.mod
	Dir      string              // absolute module root (Config.Dir)
	Packages []*Package          // documented packages, sorted by ImportPath
	Std      map[string]struct{} // import paths printed by `go list std` for this toolchain
}

// Package is one documented package.
type Package struct {
	ImportPath string
	Name       string         // package clause name
	RelDir     string         // slash-separated path relative to Module.Path; "" for the root package
	Fset       *token.FileSet // shared by all packages of the module
	Files      []*ast.File    // non-test files, sorted by filename
	Types      *types.Package
	Info       *types.Info
	Doc        *doc.Package // built with doc.NewFromFiles(Fset, Files, ImportPath), mode 0
}

// Load loads every package matching ./... under cfg.Dir and returns the documented subset.
func Load(ctx context.Context, cfg Config) (*Module, error) {
	data, err := os.ReadFile(filepath.Join(cfg.Dir, "go.mod"))
	if errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("no go.mod in %s", cfg.Dir)
	}
	if err != nil {
		return nil, fmt.Errorf("read go.mod: %w", err)
	}
	modulePath := modfile.ModulePath(data)
	if modulePath == "" {
		return nil, fmt.Errorf("go.mod in %s has no module directive", cfg.Dir)
	}

	env := append(os.Environ(), "GOOS="+cfg.GOOS, "GOARCH="+cfg.GOARCH, "CGO_ENABLED=0", "GOWORK=off")

	pcfg := &packages.Config{
		Context: ctx,
		Dir:     cfg.Dir,
		Env:     env,
		Tests:   false,
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles | packages.NeedSyntax |
			packages.NeedTypes | packages.NeedTypesInfo | packages.NeedImports | packages.NeedModule,
	}
	if len(cfg.Tags) > 0 {
		pcfg.BuildFlags = []string{"-tags=" + strings.Join(cfg.Tags, ",")}
	}
	pkgs, err := packages.Load(pcfg, "./...")
	if err != nil {
		return nil, fmt.Errorf("load packages: %w", err)
	}
	if err := collectErrors(pkgs); err != nil {
		return nil, err
	}

	var documented []*Package
	for _, pkg := range pkgs {
		if !keep(pkg, modulePath) {
			continue
		}
		p, err := buildPackage(pkg, modulePath)
		if err != nil {
			return nil, err
		}
		documented = append(documented, p)
	}
	slices.SortFunc(documented, func(a, b *Package) int {
		return strings.Compare(a.ImportPath, b.ImportPath)
	})

	std, err := loadStd(ctx, cfg.Dir, env)
	if err != nil {
		return nil, err
	}
	return &Module{Path: modulePath, Dir: cfg.Dir, Packages: documented, Std: std}, nil
}

// collectErrors returns one error listing every package error, deduplicated and sorted.
// A package that does not type-check would produce wrong links, so partial output is never written.
func collectErrors(pkgs []*packages.Package) error {
	seen := map[string]struct{}{}
	for _, pkg := range pkgs {
		for _, e := range pkg.Errors {
			seen[e.Error()] = struct{}{}
		}
	}
	if len(seen) == 0 {
		return nil
	}
	msgs := make([]string, 0, len(seen))
	for m := range seen {
		msgs = append(msgs, m)
	}
	slices.Sort(msgs)
	return errors.New(strings.Join(msgs, "\n"))
}

func relDir(pkgPath, modulePath string) string {
	return strings.TrimPrefix(strings.TrimPrefix(pkgPath, modulePath), "/")
}

func keep(pkg *packages.Package, modulePath string) bool {
	if pkg.Module == nil || pkg.Module.Path != modulePath {
		return false
	}
	if pkg.Name == "main" || len(pkg.Syntax) == 0 {
		return false
	}
	return !slices.Contains(strings.Split(relDir(pkg.PkgPath, modulePath), "/"), "internal")
}

func buildPackage(pkg *packages.Package, modulePath string) (*Package, error) {
	files := slices.Clone(pkg.Syntax)
	slices.SortFunc(files, func(a, b *ast.File) int {
		return strings.Compare(pkg.Fset.File(a.Pos()).Name(), pkg.Fset.File(b.Pos()).Name())
	})
	// Mode 0 removes unexported declarations and fields. types.Info stays valid because go/doc edits the
	// same nodes in place.
	d, err := doc.NewFromFiles(pkg.Fset, files, pkg.PkgPath)
	if err != nil {
		return nil, fmt.Errorf("build doc for %s: %w", pkg.PkgPath, err)
	}
	return &Package{
		ImportPath: pkg.PkgPath,
		Name:       pkg.Name,
		RelDir:     relDir(pkg.PkgPath, modulePath),
		Fset:       pkg.Fset,
		Files:      files,
		Types:      pkg.Types,
		Info:       pkg.TypesInfo,
		Doc:        d,
	}, nil
}

func loadStd(ctx context.Context, dir string, env []string) (map[string]struct{}, error) {
	pkgs, err := packages.Load(&packages.Config{
		Context: ctx,
		Dir:     dir,
		Env:     env,
		Mode:    packages.NeedName,
	}, "std")
	if err != nil {
		return nil, fmt.Errorf("load std packages: %w", err)
	}
	std := make(map[string]struct{}, len(pkgs))
	for _, p := range pkgs {
		std[p.PkgPath] = struct{}{}
	}
	return std, nil
}
