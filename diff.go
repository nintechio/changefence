package main

import (
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
)

func parseNumstat(data string) ([]FileChange, error) {
	out := []FileChange{}
	separator := "\n"
	if strings.ContainsRune(data, '\x00') {
		separator = "\x00"
	}
	data = strings.TrimSuffix(data, separator)
	for _, line := range strings.Split(data, separator) {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 3)
		if len(parts) != 3 {
			return nil, fmt.Errorf("invalid git numstat line: %q", line)
		}
		if parts[2] == "" {
			return nil, errors.New("invalid git numstat: empty path")
		}
		change := FileChange{Path: parts[2], Status: "modified"}
		if parts[0] == "-" || parts[1] == "-" {
			if parts[0] != "-" || parts[1] != "-" {
				return nil, fmt.Errorf("invalid binary numstat for %q", parts[2])
			}
			change.Binary = true
		} else {
			var err error
			change.Additions, err = strconv.Atoi(parts[0])
			if err != nil {
				return nil, fmt.Errorf("invalid additions for %q: %w", parts[2], err)
			}
			change.Deletions, err = strconv.Atoi(parts[1])
			if err != nil {
				return nil, fmt.Errorf("invalid deletions for %q: %w", parts[2], err)
			}
			if change.Additions < 0 || change.Deletions < 0 {
				return nil, fmt.Errorf("invalid negative numstat for %q", parts[2])
			}
		}
		out = append(out, change)
	}
	return out, nil
}

func evaluate(policy Policy, base string, files []FileChange, approvals []string) Report {
	report := Report{Version: version, Base: base, Files: files, Violations: []Violation{}}
	add := func(v Violation) {
		v.Approved = approved(v, approvals)
		report.Violations = append(report.Violations, v)
	}
	if policy.MaxFiles > 0 && len(files) > policy.MaxFiles {
		add(Violation{Rule: "max-files", Message: fmt.Sprintf("changed %d files; budget is %d", len(files), policy.MaxFiles)})
	}
	changedSource, changedTest := false, false
	for _, file := range files {
		report.Additions += file.Additions
		report.Deletions += file.Deletions
		if len(policy.Allowed) > 0 && !matchesAny(policy.Allowed, file.Path) {
			add(Violation{Rule: "outside-allowed", Path: file.Path, Message: "path is outside the allowed change scope"})
		}
		if matchesAny(policy.Forbidden, file.Path) {
			add(Violation{Rule: "forbidden", Path: file.Path, Message: "path is forbidden"})
		}
		if matchesAny(policy.Protected, file.Path) {
			add(Violation{Rule: "protected", Path: file.Path, Message: "protected path requires separate human review"})
		}
		if policy.MaxLinesPerFile > 0 && file.Additions+file.Deletions > policy.MaxLinesPerFile {
			add(Violation{Rule: "max-lines-per-file", Path: file.Path, Message: fmt.Sprintf("changed %d lines; per-file budget is %d", file.Additions+file.Deletions, policy.MaxLinesPerFile)})
		}
		if !policy.AllowDependencies && matchesAny(dependencyFiles, filepath.Base(file.Path)) {
			add(Violation{Rule: "dependency-change", Path: file.Path, Message: "dependency manifest changed but dependency changes are not allowed"})
		}
		if !policy.AllowBinary && file.Binary {
			add(Violation{Rule: "binary-change", Path: file.Path, Message: "binary change cannot be measured as line churn"})
		}
		changedSource = changedSource || matchesAny(policy.SourcePatterns, file.Path)
		changedTest = changedTest || matchesAny(policy.TestPatterns, file.Path)
	}
	if policy.MaxAdditions > 0 && report.Additions > policy.MaxAdditions {
		add(Violation{Rule: "max-additions", Message: fmt.Sprintf("added %d lines; budget is %d", report.Additions, policy.MaxAdditions)})
	}
	if policy.MaxDeletions > 0 && report.Deletions > policy.MaxDeletions {
		add(Violation{Rule: "max-deletions", Message: fmt.Sprintf("deleted %d lines; budget is %d", report.Deletions, policy.MaxDeletions)})
	}
	if policy.RequireTests && changedSource && !changedTest {
		add(Violation{Rule: "tests-required", Message: "source changed without a matching test-file change"})
	}
	for _, zone := range policy.Zones {
		var selected []FileChange
		zoneAdditions, zoneDeletions := 0, 0
		for _, file := range files {
			if matchesAny(zone.Patterns, file.Path) {
				selected = append(selected, file)
				zoneAdditions += file.Additions
				zoneDeletions += file.Deletions
			}
		}
		if zone.MaxFiles > 0 && len(selected) > zone.MaxFiles {
			add(Violation{Rule: "zone-max-files", Zone: zone.Name, Message: fmt.Sprintf("zone %q changed %d files; budget is %d", zone.Name, len(selected), zone.MaxFiles)})
		}
		if zone.MaxAdditions > 0 && zoneAdditions > zone.MaxAdditions {
			add(Violation{Rule: "zone-max-additions", Zone: zone.Name, Message: fmt.Sprintf("zone %q added %d lines; budget is %d", zone.Name, zoneAdditions, zone.MaxAdditions)})
		}
		if zone.MaxDeletions > 0 && zoneDeletions > zone.MaxDeletions {
			add(Violation{Rule: "zone-max-deletions", Zone: zone.Name, Message: fmt.Sprintf("zone %q deleted %d lines; budget is %d", zone.Name, zoneDeletions, zone.MaxDeletions)})
		}
		for _, file := range selected {
			if zone.MaxLinesPerFile > 0 && file.Additions+file.Deletions > zone.MaxLinesPerFile {
				add(Violation{Rule: "zone-max-lines-per-file", Zone: zone.Name, Path: file.Path, Message: fmt.Sprintf("changed %d lines in zone %q; per-file budget is %d", file.Additions+file.Deletions, zone.Name, zone.MaxLinesPerFile)})
			}
			if zone.Protected {
				add(Violation{Rule: "zone-protected", Zone: zone.Name, Path: file.Path, Message: fmt.Sprintf("zone %q requires explicit approval", zone.Name)})
			}
		}
	}
	report.Passed = true
	for _, violation := range report.Violations {
		if !violation.Approved {
			report.Passed = false
			break
		}
	}
	return report
}
