package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

type Policy struct {
	Schema            string   `json:"$schema,omitempty"`
	MaxFiles          int      `json:"max_files"`
	MaxAdditions      int      `json:"max_additions"`
	MaxDeletions      int      `json:"max_deletions"`
	MaxLinesPerFile   int      `json:"max_lines_per_file"`
	Allowed           []string `json:"allowed"`
	Forbidden         []string `json:"forbidden"`
	Protected         []string `json:"protected"`
	TestPatterns      []string `json:"test_patterns"`
	SourcePatterns    []string `json:"source_patterns"`
	RequireTests      bool     `json:"require_tests_for_source"`
	AllowDependencies bool     `json:"allow_dependency_changes"`
}

type FileChange struct {
	Path      string `json:"path"`
	Status    string `json:"status"`
	Additions int    `json:"additions"`
	Deletions int    `json:"deletions"`
	Binary    bool   `json:"binary"`
}

type Violation struct {
	Rule    string `json:"rule"`
	Path    string `json:"path,omitempty"`
	Message string `json:"message"`
}

type Report struct {
	Base       string       `json:"base"`
	Files      []FileChange `json:"files"`
	Additions  int          `json:"additions"`
	Deletions  int          `json:"deletions"`
	Violations []Violation  `json:"violations"`
}

var dependencyFiles = []string{"package.json", "pyproject.toml", "Cargo.toml", "go.mod", "Gemfile", "composer.json", "pom.xml", "build.gradle", "requirements.txt"}

func defaults() Policy {
	return Policy{MaxFiles: 12, MaxAdditions: 500, MaxDeletions: 500, MaxLinesPerFile: 300,
		Protected:      []string{".github/workflows/**", "CODEOWNERS", "SECURITY.md"},
		TestPatterns:   []string{"**/test/**", "**/tests/**", "**/*_test.*", "**/*.test.*", "**/*.spec.*"},
		SourcePatterns: []string{"src/**", "lib/**", "app/**", "cmd/**", "internal/**", "pkg/**"}, RequireTests: true}
}

func match(pattern, path string) bool {
	pattern, path = filepath.ToSlash(pattern), filepath.ToSlash(path)
	if strings.HasPrefix(pattern, "**/") {
		if ok, _ := filepath.Match(strings.TrimPrefix(pattern, "**/"), path); ok {
			return true
		}
		pattern = strings.TrimPrefix(pattern, "**/")
	}
	if strings.HasSuffix(pattern, "/**") {
		return strings.HasPrefix(path, strings.TrimSuffix(pattern, "**"))
	}
	ok, _ := filepath.Match(pattern, path)
	if ok {
		return true
	}
	// Match basename-oriented patterns such as **/*.test.* at any depth.
	ok, _ = filepath.Match(strings.TrimPrefix(pattern, "**/"), filepath.Base(path))
	return ok
}

func matchesAny(patterns []string, path string) bool {
	for _, p := range patterns {
		if match(p, path) {
			return true
		}
	}
	return false
}

func parseNumstat(data string) ([]FileChange, error) {
	var out []FileChange
	separator := "\n"
	if strings.ContainsRune(data, '\x00') {
		separator = "\x00"
	}
	for _, line := range strings.Split(strings.Trim(data, "\n\x00"), separator) {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 3)
		if len(parts) != 3 {
			return nil, fmt.Errorf("invalid git numstat line: %q", line)
		}
		change := FileChange{Path: parts[2], Status: "modified"}
		if parts[0] == "-" || parts[1] == "-" {
			change.Binary = true
		} else {
			var err error
			change.Additions, err = strconv.Atoi(parts[0])
			if err != nil {
				return nil, err
			}
			change.Deletions, err = strconv.Atoi(parts[1])
			if err != nil {
				return nil, err
			}
		}
		out = append(out, change)
	}
	return out, nil
}

func evaluate(policy Policy, base string, files []FileChange) Report {
	r := Report{Base: base, Files: files, Violations: []Violation{}}
	if policy.MaxFiles > 0 && len(files) > policy.MaxFiles {
		r.Violations = append(r.Violations, Violation{Rule: "max-files", Message: fmt.Sprintf("changed %d files; budget is %d", len(files), policy.MaxFiles)})
	}
	changedSource, changedTest := false, false
	for _, f := range files {
		r.Additions += f.Additions
		r.Deletions += f.Deletions
		if len(policy.Allowed) > 0 && !matchesAny(policy.Allowed, f.Path) {
			r.Violations = append(r.Violations, Violation{Rule: "outside-allowed", Path: f.Path, Message: "path is outside the allowed change scope"})
		}
		if matchesAny(policy.Forbidden, f.Path) {
			r.Violations = append(r.Violations, Violation{Rule: "forbidden", Path: f.Path, Message: "path is forbidden"})
		}
		if matchesAny(policy.Protected, f.Path) {
			r.Violations = append(r.Violations, Violation{Rule: "protected", Path: f.Path, Message: "protected path requires separate human review"})
		}
		if policy.MaxLinesPerFile > 0 && f.Additions+f.Deletions > policy.MaxLinesPerFile {
			r.Violations = append(r.Violations, Violation{Rule: "max-lines-per-file", Path: f.Path, Message: fmt.Sprintf("changed %d lines; per-file budget is %d", f.Additions+f.Deletions, policy.MaxLinesPerFile)})
		}
		if !policy.AllowDependencies && matchesAny(dependencyFiles, filepath.Base(f.Path)) {
			r.Violations = append(r.Violations, Violation{Rule: "dependency-change", Path: f.Path, Message: "dependency manifest changed but dependency changes are not allowed"})
		}
		changedSource = changedSource || matchesAny(policy.SourcePatterns, f.Path)
		changedTest = changedTest || matchesAny(policy.TestPatterns, f.Path)
	}
	if policy.MaxAdditions > 0 && r.Additions > policy.MaxAdditions {
		r.Violations = append(r.Violations, Violation{Rule: "max-additions", Message: fmt.Sprintf("added %d lines; budget is %d", r.Additions, policy.MaxAdditions)})
	}
	if policy.MaxDeletions > 0 && r.Deletions > policy.MaxDeletions {
		r.Violations = append(r.Violations, Violation{Rule: "max-deletions", Message: fmt.Sprintf("deleted %d lines; budget is %d", r.Deletions, policy.MaxDeletions)})
	}
	if policy.RequireTests && changedSource && !changedTest {
		r.Violations = append(r.Violations, Violation{Rule: "tests-required", Message: "source changed without a matching test-file change"})
	}
	return r
}

func loadPolicy(path string) (Policy, error) {
	p := defaults()
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return p, nil
	}
	if err != nil {
		return p, err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&p); err != nil {
		return p, fmt.Errorf("parse %s: %w", path, err)
	}
	return p, nil
}

func gitNumstat(base string) (string, error) {
	cmd := exec.Command("git", "-c", "core.quotePath=false", "diff", "--numstat", "-z", "--no-renames", base+"...HEAD")
	b, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git diff: %s", strings.TrimSpace(string(b)))
	}
	return string(b), nil
}

func run(args []string) int {
	fs := flag.NewFlagSet("changefence", flag.ContinueOnError)
	base := fs.String("base", "origin/main", "base revision to compare with HEAD")
	config := fs.String("config", ".changefence.json", "policy file")
	format := fs.String("format", "text", "output: text or json")
	input := fs.String("numstat", "", "read git --numstat data from a file (testing/offline)")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	p, err := loadPolicy(*config)
	if err != nil {
		fmt.Fprintln(os.Stderr, "changefence:", err)
		return 2
	}
	var data string
	if *input != "" {
		b, e := os.ReadFile(*input)
		err, data = e, string(b)
	} else {
		data, err = gitNumstat(*base)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "changefence:", err)
		return 2
	}
	files, err := parseNumstat(data)
	if err != nil {
		fmt.Fprintln(os.Stderr, "changefence:", err)
		return 2
	}
	report := evaluate(p, *base, files)
	if *format == "json" {
		b, _ := json.MarshalIndent(report, "", "  ")
		fmt.Println(string(b))
	} else {
		fmt.Printf("changefence: %d files, +%d/-%d\n", len(files), report.Additions, report.Deletions)
		for _, v := range report.Violations {
			where := ""
			if v.Path != "" {
				where = " (" + v.Path + ")"
			}
			fmt.Printf("  FAIL %-20s %s%s\n", v.Rule, v.Message, where)
		}
		if len(report.Violations) == 0 {
			fmt.Println("  PASS change is within policy")
		}
	}
	if len(report.Violations) > 0 {
		return 1
	}
	return 0
}

func main() { os.Exit(run(os.Args[1:])) }
