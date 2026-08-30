package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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
	files, err := parseNumstat("1\t0\tdocs/name with space.md\x002\t1\tsrc/café.go\x003\t0\tdocs/trailing-newline.md\n\x00")
	if err != nil || len(files) != 3 {
		t.Fatalf("parse: %#v %v", files, err)
	}
	if files[0].Path != "docs/name with space.md" || files[1].Path != "src/café.go" || files[2].Path != "docs/trailing-newline.md\n" {
		t.Fatalf("paths changed: %#v", files)
	}
}

func TestParseNumstatFailsClosed(t *testing.T) {
	for _, input := range []string{"-1\t0\tbad", "1\t-\tbad", "1\t0\t"} {
		if _, err := parseNumstat(input); err == nil {
			t.Errorf("accepted %q", input)
		}
	}
	files, err := parseNumstat("")
	if err != nil || files == nil || len(files) != 0 {
		t.Fatalf("empty diff should be an empty list: %#v %v", files, err)
	}
}

func TestEvaluateCatchesScopeAndMissingTests(t *testing.T) {
	p := defaults()
	p.Allowed = []string{"src/**", "tests/**"}
	p.MaxLinesPerFile = 10
	r := evaluate(p, "main", []FileChange{{Path: "src/app.go", Additions: 12}, {Path: "docs/x.md", Additions: 1}}, nil)
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
	r := evaluate(p, "main", []FileChange{{Path: "src/app.go", Additions: 4}, {Path: "src/app_test.go", Additions: 8}}, nil)
	if len(r.Violations) != 0 {
		t.Fatalf("unexpected: %#v", r.Violations)
	}
	if !r.Passed {
		t.Fatal("report should pass")
	}
}

func TestProtectedAndDependencyChanges(t *testing.T) {
	p := defaults()
	r := evaluate(p, "main", []FileChange{{Path: ".github/workflows/ci.yml", Additions: 2}, {Path: "package.json", Additions: 1}}, nil)
	if len(r.Violations) != 2 {
		t.Fatalf("expected two violations: %#v", r.Violations)
	}
}

func TestBinaryChangesFailClosedByDefault(t *testing.T) {
	p := defaults()
	r := evaluate(p, "main", []FileChange{{Path: "assets/archive.bin", Binary: true}}, nil)
	if r.Passed || len(r.Violations) != 1 || r.Violations[0].Rule != "binary-change" {
		t.Fatalf("binary change was not rejected: %#v", r)
	}
	p.AllowBinary = true
	r = evaluate(p, "main", []FileChange{{Path: "assets/archive.bin", Binary: true}}, nil)
	if !r.Passed {
		t.Fatalf("explicitly allowed binary change failed: %#v", r)
	}
}

func TestExplicitApprovalIsReportedAndPasses(t *testing.T) {
	p := defaults()
	r := evaluate(p, "main", []FileChange{{Path: ".github/workflows/ci.yml", Additions: 2}}, []string{"protected:.github/workflows/**"})
	if !r.Passed || len(r.Violations) != 1 || !r.Violations[0].Approved {
		t.Fatalf("approval not applied: %#v", r)
	}
}

func TestForbiddenPathCannotBeApproved(t *testing.T) {
	p := defaults()
	p.Forbidden = []string{"secrets/**"}
	r := evaluate(p, "main", []FileChange{{Path: "secrets/prod.key", Additions: 1}}, []string{"forbidden"})
	if r.Passed || len(r.Violations) != 1 || r.Violations[0].Approved {
		t.Fatalf("forbidden path was approved: %#v", r)
	}
}

func TestZoneBudgetsAndScopedApproval(t *testing.T) {
	p := defaults()
	p.Zones = []Zone{{Name: "migrations", Patterns: []string{"db/migrations/**"}, MaxFiles: 1, Protected: true}}
	files := []FileChange{{Path: "db/migrations/001.sql", Additions: 2}, {Path: "db/migrations/002.sql", Additions: 2}}
	r := evaluate(p, "main", files, []string{"zone-max-files@migrations", "zone-protected@migrations:db/migrations/**"})
	if !r.Passed || len(r.Violations) != 3 {
		t.Fatalf("zone approvals not applied: %#v", r)
	}
	for _, v := range r.Violations {
		if !v.Approved {
			t.Fatalf("unapproved violation: %#v", v)
		}
	}
}

func TestPolicyValidationRejectsInvalidValues(t *testing.T) {
	p := defaults()
	p.MaxFiles = -1
	if err := validatePolicy(p); err == nil {
		t.Fatal("negative limit accepted")
	}
	p = defaults()
	p.Forbidden = []string{"["}
	if err := validatePolicy(p); err == nil {
		t.Fatal("invalid glob accepted")
	}
	p = defaults()
	p.Zones = []Zone{{Name: "duplicate", Patterns: []string{"a/**"}}, {Name: "duplicate", Patterns: []string{"b/**"}}}
	if err := validatePolicy(p); err == nil {
		t.Fatal("duplicate zone accepted")
	}
	p = defaults()
	p.Allowed = []string{"src//**"}
	if err := validatePolicy(p); err == nil {
		t.Fatal("glob with empty segment accepted")
	}
}

func TestParseNameStatus(t *testing.T) {
	statuses, err := parseNameStatus("A\x00new.go\x00D\x00old.go\x00")
	if err != nil {
		t.Fatal(err)
	}
	if statuses["new.go"] != "added" || statuses["old.go"] != "deleted" {
		t.Fatalf("wrong statuses: %#v", statuses)
	}
	if _, err := parseNameStatus("M\x00"); err == nil {
		t.Fatal("empty path accepted")
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

func TestRepositoryPoliciesLoad(t *testing.T) {
	for _, policyPath := range []string{".changefence.json", ".changefence.example.json"} {
		if _, err := loadPolicy(policyPath); err != nil {
			t.Errorf("%s: %v", policyPath, err)
		}
	}
}

func TestGlobSemantics(t *testing.T) {
	cases := []struct {
		pattern, path string
		want          bool
	}{
		{"src/**", "src/a/b.go", true}, {"**/*_test.*", "pkg/a/b_test.go", true},
		{"**/tests/**", "packages/api/tests/unit/app_test.go", true}, {"app/**/generated/*.go", "app/api/v1/generated/types.go", true},
		{"CODEOWNERS", ".github/CODEOWNERS", true}, {"src/**", "source/a.go", false},
	}
	for _, tc := range cases {
		if got := match(tc.pattern, tc.path); got != tc.want {
			t.Errorf("match(%q,%q)=%v", tc.pattern, tc.path, got)
		}
	}
}

func TestBasePolicyCannotBeWeakenedByWorktree(t *testing.T) {
	dir := t.TempDir()
	runGit(t, dir, "init", "-q")
	policyPath := filepath.Join(dir, ".changefence.json")
	if err := os.WriteFile(policyPath, []byte(`{"max_files":1}`), 0600); err != nil {
		t.Fatal(err)
	}
	runGit(t, dir, "add", ".changefence.json")
	runGit(t, dir, "-c", "user.name=Changefence Tests", "-c", "user.email=tests@example.invalid", "commit", "-qm", "base policy")
	if err := os.WriteFile(policyPath, []byte(`{"max_files":999}`), 0600); err != nil {
		t.Fatal(err)
	}

	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(old) })
	p, source, err := loadBasePolicy("HEAD", ".changefence.json")
	if err != nil {
		t.Fatal(err)
	}
	if p.MaxFiles != 1 || !strings.HasPrefix(source, "base:") {
		t.Fatalf("loaded untrusted policy: max=%d source=%q", p.MaxFiles, source)
	}
}

func TestBaseConfigPathCannotEscapeRepository(t *testing.T) {
	for _, input := range []string{"", "../policy.json", "/tmp/policy.json", "policies/*.json", ":(glob)policy.json", `dir\policy.json`, "policy\n.json"} {
		if _, err := validateConfigPath(input); err == nil {
			t.Errorf("accepted %q", input)
		}
	}
}

func TestReportRenderingEscapesUntrustedPaths(t *testing.T) {
	report := Report{
		Version: "test", PolicyPath: ".changefence.json", PolicySource: "base:abc", Passed: false,
		Violations: []Violation{{Rule: "protected", Path: "docs/file\n::error::injected", Message: "protected path"}},
	}
	var textOutput bytes.Buffer
	renderText(&textOutput, report)
	if strings.Contains(textOutput.String(), "\n::error::injected") {
		t.Fatalf("text output contains a raw control character: %q", textOutput.String())
	}
	var githubOutput bytes.Buffer
	renderGitHub(&githubOutput, report)
	if strings.Contains(githubOutput.String(), "file=docs/file\n") || !strings.Contains(githubOutput.String(), "%0A") {
		t.Fatalf("GitHub annotation was not escaped: %q", githubOutput.String())
	}
}

func TestReportJSONUsesEmptyArrays(t *testing.T) {
	r := evaluate(defaults(), "main", []FileChange{}, nil)
	b, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(b, []byte(`"files":[]`)) || !bytes.Contains(b, []byte(`"violations":[]`)) {
		t.Fatalf("report arrays are not stable: %s", b)
	}
}

func TestGitHubPullRequestBaseComesFromEventPayload(t *testing.T) {
	eventPath := filepath.Join(t.TempDir(), "event.json")
	if err := os.WriteFile(eventPath, []byte(`{"pull_request":{"base":{"sha":"0123456789abcdef"}}}`), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GITHUB_EVENT_NAME", "pull_request")
	t.Setenv("GITHUB_EVENT_PATH", eventPath)
	base, err := githubEventBase()
	if err != nil {
		t.Fatal(err)
	}
	if base != "0123456789abcdef" {
		t.Fatalf("wrong base: %q", base)
	}
}

func TestGitHubPullRequestEventFailsClosed(t *testing.T) {
	t.Setenv("GITHUB_EVENT_NAME", "pull_request")
	t.Setenv("GITHUB_EVENT_PATH", "")
	if _, err := githubEventBase(); err == nil {
		t.Fatal("missing event payload was accepted")
	}
}

func TestRunIgnoresUntrustedBaseArgumentOnGitHubPullRequest(t *testing.T) {
	dir := t.TempDir()
	runGit(t, dir, "init", "-q")
	basePolicy := `{"max_files":1,"protected":[".changefence.json"],"require_tests_for_source":false}`
	if err := os.WriteFile(filepath.Join(dir, ".changefence.json"), []byte(basePolicy), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("base\n"), 0600); err != nil {
		t.Fatal(err)
	}
	runGit(t, dir, "add", ".")
	runGit(t, dir, "-c", "user.name=Changefence Tests", "-c", "user.email=tests@example.invalid", "commit", "-qm", "base")
	baseSHA := strings.TrimSpace(runGitOutput(t, dir, "rev-parse", "HEAD"))

	weak := `{"max_files":999,"protected":[],"require_tests_for_source":false,"allow_dependency_changes":true,"allow_binary_changes":true}`
	if err := os.WriteFile(filepath.Join(dir, ".changefence.json"), []byte(weak), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "one.md"), []byte("one\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "two.md"), []byte("two\n"), 0600); err != nil {
		t.Fatal(err)
	}
	runGit(t, dir, "add", ".")
	runGit(t, dir, "-c", "user.name=Changefence Tests", "-c", "user.email=tests@example.invalid", "commit", "-qm", "proposed weakening")

	eventPath := filepath.Join(dir, "event.json")
	event := `{"pull_request":{"base":{"sha":"` + baseSHA + `"}}}`
	if err := os.WriteFile(eventPath, []byte(event), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GITHUB_EVENT_NAME", "pull_request")
	t.Setenv("GITHUB_EVENT_PATH", eventPath)
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(old) })
	if code := run([]string{"-base", "HEAD", "-format", "json"}); code != 1 {
		t.Fatalf("untrusted HEAD policy bypassed event base: exit %d", code)
	}
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, output)
	}
}

func runGitOutput(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, output)
	}
	return string(output)
}
