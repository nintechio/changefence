package main

import (
	"errors"
	"fmt"
	"path"
	"path/filepath"
	"strings"
)

func match(pattern, name string) bool {
	pattern, name = filepath.ToSlash(pattern), filepath.ToSlash(name)
	if !strings.Contains(pattern, "/") {
		ok, _ := path.Match(pattern, path.Base(name))
		return ok
	}
	patternParts, nameParts := strings.Split(pattern, "/"), strings.Split(name, "/")
	type state struct{ pattern, name int }
	memo := map[state]bool{}
	seen := map[state]bool{}
	var visit func(int, int) bool
	visit = func(pi, ni int) bool {
		key := state{pi, ni}
		if seen[key] {
			return memo[key]
		}
		seen[key] = true
		if pi == len(patternParts) {
			memo[key] = ni == len(nameParts)
			return memo[key]
		}
		if patternParts[pi] == "**" {
			memo[key] = visit(pi+1, ni) || (ni < len(nameParts) && visit(pi, ni+1))
			return memo[key]
		}
		if ni < len(nameParts) {
			segmentMatch, _ := path.Match(patternParts[pi], nameParts[ni])
			memo[key] = segmentMatch && visit(pi+1, ni+1)
		}
		return memo[key]
	}
	return visit(0, 0)
}

func matchesAny(patterns []string, name string) bool {
	for _, pattern := range patterns {
		if match(pattern, name) {
			return true
		}
	}
	return false
}

func validatePolicy(p Policy) error {
	limits := []struct {
		name  string
		value int
	}{
		{"max_files", p.MaxFiles},
		{"max_additions", p.MaxAdditions},
		{"max_deletions", p.MaxDeletions},
		{"max_lines_per_file", p.MaxLinesPerFile},
	}
	for _, limit := range limits {
		if limit.value < 0 {
			return fmt.Errorf("%s must be zero or positive", limit.name)
		}
	}
	patternGroups := []struct {
		name     string
		patterns []string
	}{
		{"allowed", p.Allowed},
		{"forbidden", p.Forbidden},
		{"protected", p.Protected},
		{"test_patterns", p.TestPatterns},
		{"source_patterns", p.SourcePatterns},
	}
	seenZones := map[string]bool{}
	for i, zone := range p.Zones {
		if strings.TrimSpace(zone.Name) == "" {
			return fmt.Errorf("zones[%d].name must not be empty", i)
		}
		if seenZones[zone.Name] {
			return fmt.Errorf("duplicate zone name %q", zone.Name)
		}
		seenZones[zone.Name] = true
		if len(zone.Patterns) == 0 {
			return fmt.Errorf("zone %q must contain at least one pattern", zone.Name)
		}
		patternGroups = append(patternGroups, struct {
			name     string
			patterns []string
		}{fmt.Sprintf("zones[%d].patterns", i), zone.Patterns})
		zoneLimits := []int{zone.MaxFiles, zone.MaxAdditions, zone.MaxDeletions, zone.MaxLinesPerFile}
		for _, value := range zoneLimits {
			if value < 0 {
				return fmt.Errorf("zone %q limits must be zero or positive", zone.Name)
			}
		}
	}
	for _, group := range patternGroups {
		for i, pattern := range group.patterns {
			if strings.TrimSpace(pattern) == "" {
				return fmt.Errorf("%s[%d] must not be empty", group.name, i)
			}
			if err := validateGlob(pattern); err != nil {
				return fmt.Errorf("%s[%d] invalid glob %q: %w", group.name, i, pattern, err)
			}
		}
	}
	return nil
}

func validateGlob(pattern string) error {
	if strings.Contains(pattern, "\\") {
		return errors.New("use forward slashes in globs")
	}
	for _, segment := range strings.Split(filepath.ToSlash(pattern), "/") {
		if segment == "" || segment == "." || segment == ".." {
			return errors.New("glob must be repository-relative without empty, dot, or parent segments")
		}
		if segment == "**" {
			continue
		}
		if _, err := path.Match(segment, segment); err != nil {
			return err
		}
	}
	return nil
}

func approved(v Violation, approvals []string) bool {
	if v.Rule == "forbidden" {
		return false
	}
	for _, item := range approvals {
		selector := strings.SplitN(item, ":", 2)
		target := strings.SplitN(selector[0], "@", 2)
		if target[0] != v.Rule {
			continue
		}
		if len(target) == 2 && target[1] != v.Zone {
			continue
		}
		if len(selector) == 1 || (v.Path != "" && match(selector[1], v.Path)) {
			return true
		}
	}
	return false
}
