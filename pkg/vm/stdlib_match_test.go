package vm

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// The last arm of a `match` is not a catch-all: only `_` is. A `match` over a
// non-boolean subject whose last arm is the literal `true` therefore fails at run
// time as soon as no earlier arm matches - which is what `core/io.ca`'s
// `_format_helper` and `_replace_first` did, relying on the old lenient behaviour.
//
// Two forms are legitimate and are skipped: `match true`, where the subject *is* the
// literal, and a match whose arms are all boolean (`true`/`false`/`_`), where the
// subject is a boolean expression. What is flagged is a `true` last arm alongside a
// numeric, string or constructor pattern - arms that can only be decided by the
// subject's value, so `true` cannot stand in for a catch-all.
// hasNonBooleanArm reports whether any arm is a numeric literal, a string literal or
// a constructor pattern - the shapes whose match cannot be decided by a `true` arm.
func hasNonBooleanArm(arms []string) bool {
	for _, a := range arms {
		if a == "true" || a == "false" || a == "_" {
			continue
		}
		// A numeric literal, a string literal, or a pattern with a payload
		// (`Some(x)`, `success(v)`, `-1`).
		if regexp.MustCompile(`^-?[0-9]`).MatchString(a) ||
			strings.HasPrefix(a, `"`) ||
			strings.Contains(a, "(") {
			return true
		}
	}
	return false
}

func TestStdlibMatchArmsAreExplicit(t *testing.T) {
	files, err := filepath.Glob("stdlib/core/*.ca")
	if err != nil || len(files) == 0 {
		t.Fatalf("no stdlib sources found: %v", err)
	}

	matchRE := regexp.MustCompile(`(?:^\s*|[|(=>]\s*)match\b(.*)$`)
	armRE := regexp.MustCompile(`^\s+(\S.*?)\s*=>`)

	for _, path := range files {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		lines := strings.Split(string(raw), "\n")
		for i, line := range lines {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "//") {
				continue
			}
			m := matchRE.FindStringSubmatch(line)
			if m == nil {
				continue
			}
			subject := strings.TrimSpace(m[1])
			if subject == "true" || subject == "false" {
				continue
			}
			indent := len(line) - len(strings.TrimLeft(line, " \t"))

			var arms []string
			for j := i + 1; j < len(lines); j++ {
				l := lines[j]
				if strings.TrimSpace(l) != "" &&
					len(l)-len(strings.TrimLeft(l, " \t")) <= indent {
					break
				}
				if a := armRE.FindStringSubmatch(l); a != nil {
					arms = append(arms, a[1])
				}
			}
			if len(arms) == 0 {
				continue
			}
			last := arms[len(arms)-1]
			if last != "true" || !hasNonBooleanArm(arms) {
				continue
			}
			t.Errorf("%s:%d: `match %s` ends with `true =>`; use `_ =>` (only `_` is a catch-all)",
				path, i+1, subject)
		}
	}
}
