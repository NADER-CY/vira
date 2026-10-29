package archlint

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const mod = "github.com/NADER-CY/vira/backend"

// TestCodebaseRespectsBoundaries runs the lint over the real backend module.
func TestCodebaseRespectsBoundaries(t *testing.T) {
	root, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		t.Fatalf("backend root not found at %s", root)
	}
	vs, err := Check(root, mod)
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range vs {
		t.Errorf("architecture violation: %s", v)
	}
}

// TestLintCatchesForbiddenImports proves each rule fires on a synthetic tree.
func TestLintCatchesForbiddenImports(t *testing.T) {
	root := t.TempDir()
	write := func(rel, imports string) {
		p := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		pkg := filepath.Base(filepath.Dir(p))
		pkg = strings.ReplaceAll(pkg, "-", "")
		src := "package " + pkg + "\n\nimport (\n" + imports + ")\n"
		if err := os.WriteFile(p, []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	q := func(p string) string { return "\t_ \"" + mod + "/" + p + "\"\n" }
	write("internal/platform/db/x.go", q("internal/modules/orders"))              // R1
	write("internal/modules/orders/x.go", q("internal/modules/catalog/store"))    // R2
	write("internal/modules/orders/y.go", q("internal/modules/catalog/contract")) // allowed
	write("internal/modules/orders/z.go", q("internal/app/publicapi"))            // R3
	write("internal/app/publicapi/x.go", q("internal/platform/adminauth"))        // R4
	write("cmd/admin-api/x.go", q("internal/platform/adminauth"))                 // allowed
	write("internal/modules/social/x.go", q("internal/platform/testdb"))          // R5
	write("internal/modules/social/x_test.go", q("internal/platform/testdb"))     // allowed in tests

	vs, err := Check(root, mod)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{"R1": false, "R2": false, "R3": false, "R4": false, "R5": false}
	for _, v := range vs {
		want[v.Rule[:2]] = true
	}
	for r, hit := range want {
		if !hit {
			t.Errorf("rule %s did not fire; violations: %v", r, vs)
		}
	}
	if len(vs) != 5 {
		t.Errorf("expected exactly 5 violations (allowed imports must pass), got %d: %v", len(vs), vs)
	}
}
