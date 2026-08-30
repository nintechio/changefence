package main

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
	AllowBinary       bool     `json:"allow_binary_changes"`
	Zones             []Zone   `json:"zones"`
}

type Zone struct {
	Name            string   `json:"name"`
	Patterns        []string `json:"patterns"`
	MaxFiles        int      `json:"max_files"`
	MaxAdditions    int      `json:"max_additions"`
	MaxDeletions    int      `json:"max_deletions"`
	MaxLinesPerFile int      `json:"max_lines_per_file"`
	Protected       bool     `json:"protected"`
}

type FileChange struct {
	Path      string `json:"path"`
	Status    string `json:"status"`
	Additions int    `json:"additions"`
	Deletions int    `json:"deletions"`
	Binary    bool   `json:"binary"`
}

type Violation struct {
	Rule     string `json:"rule"`
	Path     string `json:"path,omitempty"`
	Message  string `json:"message"`
	Zone     string `json:"zone,omitempty"`
	Approved bool   `json:"approved"`
}

type Report struct {
	Version      string       `json:"version"`
	Base         string       `json:"base"`
	PolicyPath   string       `json:"policy_path"`
	PolicySource string       `json:"policy_source"`
	Files        []FileChange `json:"files"`
	Additions    int          `json:"additions"`
	Deletions    int          `json:"deletions"`
	Violations   []Violation  `json:"violations"`
	Passed       bool         `json:"passed"`
}

var dependencyFiles = []string{
	"package.json", "package-lock.json", "npm-shrinkwrap.json", "pnpm-lock.yaml", "yarn.lock", "bun.lock", "bun.lockb",
	"pyproject.toml", "uv.lock", "poetry.lock", "Pipfile", "Pipfile.lock", "requirements*.txt",
	"go.mod", "go.sum", "Cargo.toml", "Cargo.lock", "Gemfile", "Gemfile.lock", "composer.json", "composer.lock",
	"pom.xml", "build.gradle", "build.gradle.kts", "settings.gradle", "settings.gradle.kts", "gradle.lockfile",
	"mix.exs", "mix.lock", "Package.swift", "Package.resolved", "Directory.Packages.props", "packages.lock.json", "*.csproj",
}

var version = "dev"

func defaults() Policy {
	return Policy{MaxFiles: 12, MaxAdditions: 500, MaxDeletions: 500, MaxLinesPerFile: 300,
		Protected:      []string{".changefence.json", ".github/workflows/**", "CODEOWNERS", "SECURITY.md"},
		TestPatterns:   []string{"**/test/**", "**/tests/**", "**/*_test.*", "**/*.test.*", "**/*.spec.*"},
		SourcePatterns: []string{"src/**", "lib/**", "app/**", "cmd/**", "internal/**", "pkg/**"}, RequireTests: true}
}
