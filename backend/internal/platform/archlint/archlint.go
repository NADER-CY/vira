// Package archlint enforces the modular-monolith boundaries (ADR-001, doc 12 P1: "architecture lint blocks a
// forbidden import"). It parses every Go file's imports and applies the rules below; the test in this package
// runs it over the real code base on every CI build.
//
//	R1 platform code never imports modules or app wiring (platform is the bottom layer).
//	R2 a module imports another module only through that module's `contract` package.
//	R3 modules never import app wiring or binaries.
//	R4 only the admin-api binary and its wiring import the admin realm (Super Admin isolation, ADR-016); tests are exempt.
//	R5 production code never imports test helpers (testdb).
package archlint

import (
	"fmt"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// Violation is one forbidden import.
type Violation struct {
	File   string
	Import string
	Rule   string
}

func (v Violation) String() string {
	return fmt.Sprintf("%s imports %s (%s)", v.File, v.Import, v.Rule)
}

// Check scans root (the backend module directory) whose module path is modPath.
func Check(root, modPath string) ([]Violation, error) {
	var out []Violation
	fset := token.NewFileSet()
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if n := d.Name(); n == "testdata" || n == "vendor" || strings.HasPrefix(n, ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		rel = filepath.ToSlash(rel)
		f, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		isTest := strings.HasSuffix(rel, "_test.go")
		for _, imp := range f.Imports {
			p, _ := strconv.Unquote(imp.Path.Value)
			if !strings.HasPrefix(p, modPath+"/") {
				continue
			}
			target := strings.TrimPrefix(p, modPath+"/")
			if rule := violates(filepath.ToSlash(filepath.Dir(rel)), target, isTest); rule != "" {
				out = append(out, Violation{File: rel, Import: target, Rule: rule})
			}
		}
		return nil
	})
	sort.Slice(out, func(i, j int) bool { return out[i].String() < out[j].String() })
	return out, err
}

func under(p, prefix string) bool { return p == prefix || strings.HasPrefix(p, prefix+"/") }

// moduleOf returns the module name for internal/modules/<name>/... paths.
func moduleOf(p string) string {
	if !under(p, "internal/modules") {
		return ""
	}
	parts := strings.Split(p, "/")
	if len(parts) < 3 {
		return ""
	}
	return parts[2]
}

func violates(fromDir, target string, isTest bool) string {
	switch {
	case under(fromDir, "internal/platform") && (under(target, "internal/modules") || under(target, "internal/app")):
		return "R1 platform must not depend on modules or app wiring"
	}
	if from := moduleOf(fromDir); from != "" {
		if to := moduleOf(target); to != "" && to != from {
			if !under(target, "internal/modules/"+to+"/contract") {
				return "R2 cross-module imports must use the target module's contract package"
			}
		}
		if under(target, "internal/app") || under(target, "cmd") {
			return "R3 modules must not depend on app wiring or binaries"
		}
	}
	// Tests may import the admin realm (e.g. the OpenAPI contract test); shipped code may not.
	if (under(target, "internal/platform/adminauth") || under(target, "internal/app/adminapi")) && !isTest {
		if !under(fromDir, "cmd/admin-api") && !under(fromDir, "internal/app/adminapi") && !under(fromDir, "internal/platform/adminauth") {
			return "R4 the admin realm is only reachable from the admin-api binary"
		}
	}
	if under(target, "internal/platform/testdb") && !isTest {
		return "R5 production code must not import test helpers"
	}
	return ""
}
