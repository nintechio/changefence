package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
)

type stringList []string

func (s *stringList) String() string { return strings.Join(*s, ",") }

func (s *stringList) Set(value string) error {
	item := strings.TrimSpace(value)
	if item == "" {
		return errors.New("approval must not be empty")
	}
	selector := strings.SplitN(item, ":", 2)
	target := strings.SplitN(selector[0], "@", 2)
	if strings.Count(selector[0], "@") > 1 || target[0] == "" || (len(target) == 2 && target[1] == "") || (len(selector) == 2 && selector[1] == "") {
		return fmt.Errorf("invalid approval %q", item)
	}
	if len(selector) == 2 {
		if err := validateGlob(selector[1]); err != nil {
			return fmt.Errorf("invalid approval %q: %w", item, err)
		}
	}
	*s = append(*s, item)
	return nil
}

func run(args []string) int {
	flags := flag.NewFlagSet("changefence", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	base := flags.String("base", "origin/main", "base revision to compare with HEAD")
	config := flags.String("config", ".changefence.json", "policy file")
	policySource := flags.String("policy-source", "auto", "policy source: auto, base, or worktree")
	format := flags.String("format", "text", "output: text, json, markdown, or github")
	input := flags.String("numstat", "", "read git --numstat data from a file (testing/offline)")
	showVersion := flags.Bool("version", false, "print version and exit")
	var approvals stringList
	flags.Var(&approvals, "approve", "approve a trusted violation selector (repeatable)")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if *showVersion {
		fmt.Println(version)
		return 0
	}
	validFormats := map[string]bool{"text": true, "json": true, "markdown": true, "github": true}
	if !validFormats[*format] {
		fmt.Fprintf(os.Stderr, "changefence: unsupported format %q\n", *format)
		return 2
	}
	sourceMode := *policySource
	if sourceMode == "auto" {
		if *input != "" {
			sourceMode = "worktree"
		} else {
			sourceMode = "base"
		}
	}
	if sourceMode != "base" && sourceMode != "worktree" {
		fmt.Fprintf(os.Stderr, "changefence: unsupported policy source %q\n", *policySource)
		return 2
	}
	effectiveBase := *base
	if eventBase, eventErr := githubEventBase(); eventErr != nil {
		fmt.Fprintln(os.Stderr, "changefence:", eventErr)
		return 2
	} else if eventBase != "" {
		effectiveBase = eventBase
	}

	policy, policyOrigin, err := selectPolicy(sourceMode, effectiveBase, *config)
	if err != nil {
		fmt.Fprintln(os.Stderr, "changefence:", err)
		return 2
	}
	files, reportBase, err := selectChanges(*input, effectiveBase)
	if err != nil {
		fmt.Fprintln(os.Stderr, "changefence:", err)
		return 2
	}

	report := evaluate(policy, reportBase, files, approvals)
	report.PolicyPath = *config
	report.PolicySource = policyOrigin
	switch *format {
	case "json":
		encoded, _ := json.MarshalIndent(report, "", "  ")
		fmt.Println(string(encoded))
	case "markdown":
		renderMarkdown(os.Stdout, report)
	case "github":
		renderGitHub(os.Stdout, report)
	default:
		renderText(os.Stdout, report)
	}
	if !report.Passed {
		return 1
	}
	return 0
}

func selectPolicy(sourceMode, base, config string) (Policy, string, error) {
	if sourceMode == "base" {
		cleanPath, err := validateConfigPath(config)
		if err != nil {
			return Policy{}, "", err
		}
		return loadBasePolicy(base, cleanPath)
	}
	policy, err := loadPolicy(config)
	return policy, "worktree", err
}

func selectChanges(input, base string) ([]FileChange, string, error) {
	if input == "" {
		return gitChanges(base)
	}
	data, err := os.ReadFile(input)
	if err != nil {
		return nil, base, err
	}
	files, err := parseNumstat(string(data))
	return files, base, err
}

func main() { os.Exit(run(os.Args[1:])) }
