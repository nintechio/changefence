package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseNumstat(t *testing.T) {
	files, err := parseNumstat("12\t3\tsrc/app.go\n-\t-\tassets/logo.png\n")
	if err != nil || len(files) != 2 {
		t.Fatalf("parse: %#v %v", files, err)
	}
	if files[0].Additions != 12 || files[0].Deletions != 3 || !files[1].Binary {
		t.Fatalf("wrong values: %#v", files)
	}
}

func TestParseNulNumstatPreservesUnusualPath(t *testing.T) {
	files, err := parseNumstat("1\t0\tdocs/name with space.md\x002\t1\tsrc/café.go\x00")
	if err != nil || len(files) != 2 {
		t.Fatalf("parse: %#v %v", files, err)
	}
	if files[0].Path != "docs/name with space.md" || files[1].Path != "src/café.go" {
		t.Fatalf("paths changed: %#v", files)
	}
}

func TestEvaluateCatchesScopeAndMissingTests(t *testing.T) {
	p := defaults()
	p.Allowed = []string{"src/**", "tests/**"}
	p.MaxLinesPerFile = 10
	r := evaluate(p, "main", []FileChange{{Path: "src/app.go", Additions: 12}, {Path: "docs/x.md", Additions: 1}})
	rules := map[string]bool{}
	for _, v := range r.Violations {
		rules[v.Rule] = true
	}
	for _, want := range []string{"max-lines-per-file", "outside-allowed", "tests-required"} {
		if !rules[want] {
			t.Errorf("missing %s: %#v", want, r.Violations)
		}
	}
}

func TestSourceAndTestPass(t *testing.T) {
	p := defaults()
	p.AllowDependencies = true
	r := evaluate(p, "main", []FileChange{{Path: "src/app.go", Additions: 4}, {Path: "src/app_test.go", Additions: 8}})
	if len(r.Violations) != 0 {
		t.Fatalf("unexpected: %#v", r.Violations)
	}
}

func TestProtectedAndDependencyChanges(t *testing.T) {
	p := defaults()
	r := evaluate(p, "main", []FileChange{{Path: ".github/workflows/ci.yml", Additions: 2}, {Path: "package.json", Additions: 1}})
	if len(r.Violations) != 2 {
		t.Fatalf("expected two violations: %#v", r.Violations)
	}
}

func TestRunOffline(t *testing.T) {
	dir := t.TempDir()
	stats := filepath.Join(dir, "stats")
	cfg := filepath.Join(dir, "policy.json")
	os.WriteFile(stats, []byte("1\t0\tREADME.md\n"), 0600)
	os.WriteFile(cfg, []byte(`{"max_files":2,"allow_dependency_changes":true}`), 0600)
	if code := run([]string{"-numstat", stats, "-config", cfg, "-format", "json"}); code != 0 {
		t.Fatalf("exit %d", code)
	}
}

func TestMalformedPolicyFailsClosed(t *testing.T) {
	dir := t.TempDir()
	stats := filepath.Join(dir, "stats")
	cfg := filepath.Join(dir, "policy.json")
	os.WriteFile(stats, []byte("1\t0\tREADME.md\n"), 0600)
	os.WriteFile(cfg, []byte(`{"max_files":"unlimited"}`), 0600)
	if code := run([]string{"-numstat", stats, "-config", cfg}); code != 2 {
		t.Fatalf("exit %d", code)
	}
}

func TestGlobSemantics(t *testing.T) {
	cases := []struct {
		pattern, path string
		want          bool
	}{
		{"src/**", "src/a/b.go", true}, {"**/*_test.*", "pkg/a/b_test.go", true},
		{"CODEOWNERS", ".github/CODEOWNERS", true}, {"src/**", "source/a.go", false},
	}
	for _, tc := range cases {
		if got := match(tc.pattern, tc.path); got != tc.want {
			t.Errorf("match(%q,%q)=%v", tc.pattern, tc.path, got)
		}
	}
}
