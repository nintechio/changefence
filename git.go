package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func resolveCommit(ref string) (string, error) {
	cmd := gitCommand("rev-parse", "--verify", "--end-of-options", ref+"^{commit}")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("resolve base %q: %s", ref, strings.TrimSpace(string(output)))
	}
	return strings.TrimSpace(string(output)), nil
}

func parseNameStatus(data string) (map[string]string, error) {
	fields := strings.Split(strings.TrimSuffix(data, "\x00"), "\x00")
	if len(fields) == 1 && fields[0] == "" {
		return map[string]string{}, nil
	}
	if len(fields)%2 != 0 {
		return nil, errors.New("invalid git name-status output")
	}
	statuses := make(map[string]string, len(fields)/2)
	for i := 0; i < len(fields); i += 2 {
		if fields[i] == "" || fields[i+1] == "" {
			return nil, errors.New("invalid git name-status output: empty status or path")
		}
		status := map[string]string{"A": "added", "M": "modified", "D": "deleted", "T": "type-changed"}[fields[i]]
		if status == "" {
			status = strings.ToLower(fields[i])
		}
		statuses[fields[i+1]] = status
	}
	return statuses, nil
}

func gitChanges(base string) ([]FileChange, string, error) {
	sha, err := resolveCommit(base)
	if err != nil {
		return nil, "", err
	}
	rangeArg := sha + "...HEAD"
	cmd := gitCommand("-c", "core.quotePath=false", "diff", "--numstat", "-z", "--no-renames", rangeArg)
	numstat, err := cmd.CombinedOutput()
	if err != nil {
		return nil, "", fmt.Errorf("git diff: %s", strings.TrimSpace(string(numstat)))
	}
	files, err := parseNumstat(string(numstat))
	if err != nil {
		return nil, "", err
	}
	cmd = gitCommand("-c", "core.quotePath=false", "diff", "--name-status", "-z", "--no-renames", rangeArg)
	nameStatus, err := cmd.CombinedOutput()
	if err != nil {
		return nil, "", fmt.Errorf("git diff status: %s", strings.TrimSpace(string(nameStatus)))
	}
	statuses, err := parseNameStatus(string(nameStatus))
	if err != nil {
		return nil, "", err
	}
	if len(statuses) != len(files) {
		return nil, "", fmt.Errorf("git diff outputs disagree: %d numstat paths, %d status paths", len(files), len(statuses))
	}
	for i := range files {
		status, ok := statuses[files[i].Path]
		if !ok {
			return nil, "", fmt.Errorf("git diff status missing for %q", files[i].Path)
		}
		files[i].Status = status
	}
	return files, sha, nil
}

func gitCommand(args ...string) *exec.Cmd {
	if cwd, err := os.Getwd(); err == nil {
		args = append([]string{"-c", "safe.directory=" + cwd}, args...)
	}
	return exec.Command("git", args...)
}
