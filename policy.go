package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func loadPolicy(policyPath string) (Policy, error) {
	policy := defaults()
	data, err := os.ReadFile(policyPath)
	if errors.Is(err, os.ErrNotExist) {
		return policy, nil
	}
	if err != nil {
		return policy, err
	}
	if err := decodePolicy(data, &policy); err != nil {
		return policy, fmt.Errorf("parse %s: %w", policyPath, err)
	}
	if err := validatePolicy(policy); err != nil {
		return policy, fmt.Errorf("validate %s: %w", policyPath, err)
	}
	return policy, nil
}

func loadBasePolicy(base, policyPath string) (Policy, string, error) {
	policy := defaults()
	if err := validatePolicy(policy); err != nil {
		return policy, "", err
	}
	sha, err := resolveCommit(base)
	if err != nil {
		return policy, "", err
	}
	gitPath := filepath.ToSlash(policyPath)
	object := sha + ":" + gitPath
	cmd := gitCommand("ls-tree", "--full-tree", "-z", sha, "--", gitPath)
	entry, err := cmd.CombinedOutput()
	if err != nil {
		return policy, "", fmt.Errorf("locate policy in base: %s", strings.TrimSpace(string(entry)))
	}
	if len(entry) == 0 {
		return policy, "defaults (policy absent from base)", nil
	}
	metadata := strings.SplitN(strings.TrimSuffix(string(entry), "\x00"), "\t", 2)
	if len(metadata) != 2 || !strings.Contains(metadata[0], " blob ") || metadata[1] != gitPath {
		return policy, "", fmt.Errorf("policy path %q is not a regular Git blob", policyPath)
	}
	cmd = gitCommand("show", object)
	data, err := cmd.CombinedOutput()
	if err != nil {
		return policy, "", fmt.Errorf("read policy from base: %s", strings.TrimSpace(string(data)))
	}
	if err := decodePolicy(data, &policy); err != nil {
		return policy, "", fmt.Errorf("parse %s from base: %w", policyPath, err)
	}
	if err := validatePolicy(policy); err != nil {
		return policy, "", fmt.Errorf("validate %s from base: %w", policyPath, err)
	}
	return policy, "base:" + sha[:12], nil
}

func decodePolicy(data []byte, policy *Policy) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(policy); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return errors.New("trailing JSON content")
	}
	return nil
}

func validateConfigPath(configPath string) (string, error) {
	containsControl := strings.IndexFunc(configPath, func(r rune) bool { return r < 0x20 || r == 0x7f }) >= 0
	if configPath == "" || containsControl || strings.Contains(configPath, "\\") || strings.ContainsAny(configPath, "*?[") || strings.HasPrefix(configPath, ":") || filepath.IsAbs(configPath) {
		return "", errors.New("base policy path must be a non-empty repository-relative path")
	}
	clean := filepath.ToSlash(filepath.Clean(configPath))
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") {
		return "", errors.New("base policy path must stay inside the repository")
	}
	return clean, nil
}
