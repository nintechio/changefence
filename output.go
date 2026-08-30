package main

import (
	"fmt"
	"io"
	"strconv"
	"strings"
)

func renderText(w io.Writer, report Report) {
	state := "PASS"
	if !report.Passed {
		state = "FAIL"
	}
	fmt.Fprintf(w, "changefence %s: %s · %d files · +%d/-%d\n", report.Version, state, len(report.Files), report.Additions, report.Deletions)
	fmt.Fprintf(w, "policy: %s (%s)\n", safeText(report.PolicyPath), safeText(report.PolicySource))
	for _, violation := range report.Violations {
		label := "FAIL"
		if violation.Approved {
			label = "ALLOW"
		}
		where := safeText(violation.Path)
		if where == "" && violation.Zone != "" {
			where = "zone:" + safeText(violation.Zone)
		}
		if where != "" {
			where = " (" + where + ")"
		}
		fmt.Fprintf(w, "  %-5s %-25s %s%s\n", label, violation.Rule, violation.Message, where)
	}
	if len(report.Violations) == 0 {
		fmt.Fprintln(w, "  PASS  change is within policy")
	}
}

func renderMarkdown(w io.Writer, report Report) {
	state := "✅ Passed"
	if !report.Passed {
		state = "❌ Failed"
	}
	fmt.Fprintf(w, "## Changefence: %s\n\n", state)
	fmt.Fprintf(w, "%d files · **+%d/-%d** · policy `%s` from `%s`\n", len(report.Files), report.Additions, report.Deletions, markdownCell(report.PolicyPath), markdownCell(report.PolicySource))
	if len(report.Violations) == 0 {
		fmt.Fprintln(w, "\nThe change is within policy.")
		return
	}
	fmt.Fprintln(w, "\n| Result | Rule | Location | Detail |")
	fmt.Fprintln(w, "|---|---|---|---|")
	for _, violation := range report.Violations {
		result := "Fail"
		if violation.Approved {
			result = "Approved"
		}
		location := violation.Path
		if location == "" && violation.Zone != "" {
			location = "zone:" + violation.Zone
		}
		fmt.Fprintf(w, "| %s | `%s` | `%s` | %s |\n", result, markdownCell(violation.Rule), markdownCell(location), markdownCell(violation.Message))
	}
}

func renderGitHub(w io.Writer, report Report) {
	for _, violation := range report.Violations {
		level := "error"
		if violation.Approved {
			level = "notice"
		}
		properties := "title=" + annotationEscape("changefence: "+violation.Rule, true)
		if violation.Path != "" {
			properties += ",file=" + annotationEscape(violation.Path, true)
		}
		fmt.Fprintf(w, "::%s %s::%s\n", level, properties, annotationEscape(violation.Message, false))
	}
	renderText(w, report)
}

func markdownCell(value string) string {
	return strings.NewReplacer(
		"&", "&amp;", "<", "&lt;", ">", "&gt;", "`", "&#96;",
		"|", "\\|", "\r", " ", "\n", " ",
	).Replace(value)
}

func safeText(value string) string {
	if strings.IndexFunc(value, func(r rune) bool { return r < 0x20 || r == 0x7f }) >= 0 {
		return strconv.Quote(value)
	}
	return value
}

func annotationEscape(value string, property bool) string {
	value = strings.ReplaceAll(value, "%", "%25")
	value = strings.ReplaceAll(value, "\r", "%0D")
	value = strings.ReplaceAll(value, "\n", "%0A")
	if property {
		value = strings.ReplaceAll(value, ":", "%3A")
		value = strings.ReplaceAll(value, ",", "%2C")
	}
	return value
}
