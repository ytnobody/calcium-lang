package bone

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Version represents a semantic version
type Version struct {
	Major int
	Minor int
	Patch int
}

// ParseVersion parses a version string like "1.2.3" or "v1.2.3"
func ParseVersion(s string) (*Version, error) {
	s = strings.TrimPrefix(s, "v")
	parts := strings.Split(s, ".")

	if len(parts) < 1 || len(parts) > 3 {
		return nil, fmt.Errorf("invalid version format: %s", s)
	}

	v := &Version{}

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return nil, fmt.Errorf("invalid major version: %s", parts[0])
	}
	v.Major = major

	if len(parts) >= 2 {
		minor, err := strconv.Atoi(parts[1])
		if err != nil {
			return nil, fmt.Errorf("invalid minor version: %s", parts[1])
		}
		v.Minor = minor
	}

	if len(parts) >= 3 {
		// Handle patch version with possible pre-release suffix
		patchStr := strings.Split(parts[2], "-")[0]
		patch, err := strconv.Atoi(patchStr)
		if err != nil {
			return nil, fmt.Errorf("invalid patch version: %s", parts[2])
		}
		v.Patch = patch
	}

	return v, nil
}

// String returns the version string
func (v *Version) String() string {
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}

// Compare compares two versions
// Returns -1 if v < other, 0 if v == other, 1 if v > other
func (v *Version) Compare(other *Version) int {
	if v.Major != other.Major {
		if v.Major < other.Major {
			return -1
		}
		return 1
	}
	if v.Minor != other.Minor {
		if v.Minor < other.Minor {
			return -1
		}
		return 1
	}
	if v.Patch != other.Patch {
		if v.Patch < other.Patch {
			return -1
		}
		return 1
	}
	return 0
}

// VersionConstraint represents a version constraint
type VersionConstraint struct {
	Operator string   // "=", "^", "~", ">=", "<=", ">", "<", "*"
	Version  *Version // nil for "*" (any version)
}

// ParseConstraint parses a version constraint string
// Supported formats:
//   - "1.2.3"     - exact version
//   - "^1.2.3"    - compatible with 1.2.3 (>=1.2.3 <2.0.0)
//   - "~1.2.3"    - approximately 1.2.3 (>=1.2.3 <1.3.0)
//   - ">=1.2.3"   - greater than or equal
//   - "<=1.2.3"   - less than or equal
//   - ">1.2.3"    - greater than
//   - "<1.2.3"    - less than
//   - "*"         - any version
//   - "latest"    - latest version
func ParseConstraint(s string) (*VersionConstraint, error) {
	s = strings.TrimSpace(s)

	if s == "*" || s == "latest" || s == "" {
		return &VersionConstraint{Operator: "*", Version: nil}, nil
	}

	// Match operator and version
	re := regexp.MustCompile(`^(\^|~|>=|<=|>|<|=)?(.+)$`)
	matches := re.FindStringSubmatch(s)
	if matches == nil {
		return nil, fmt.Errorf("invalid constraint format: %s", s)
	}

	operator := matches[1]
	versionStr := matches[2]

	if operator == "" {
		operator = "="
	}

	version, err := ParseVersion(versionStr)
	if err != nil {
		return nil, err
	}

	return &VersionConstraint{
		Operator: operator,
		Version:  version,
	}, nil
}

// Matches checks if a version satisfies the constraint
func (c *VersionConstraint) Matches(v *Version) bool {
	if c.Version == nil {
		// "*" matches any version
		return true
	}

	cmp := v.Compare(c.Version)

	switch c.Operator {
	case "=":
		return cmp == 0
	case ">":
		return cmp > 0
	case ">=":
		return cmp >= 0
	case "<":
		return cmp < 0
	case "<=":
		return cmp <= 0
	case "^":
		// Caret: >=version, <next major
		// ^1.2.3 means >=1.2.3 <2.0.0
		// ^0.2.3 means >=0.2.3 <0.3.0 (special case for 0.x)
		if cmp < 0 {
			return false
		}
		if c.Version.Major == 0 {
			// For 0.x versions, caret is more restrictive
			return v.Major == 0 && v.Minor == c.Version.Minor
		}
		return v.Major == c.Version.Major
	case "~":
		// Tilde: >=version, <next minor
		// ~1.2.3 means >=1.2.3 <1.3.0
		if cmp < 0 {
			return false
		}
		return v.Major == c.Version.Major && v.Minor == c.Version.Minor
	default:
		return false
	}
}

// String returns the constraint string
func (c *VersionConstraint) String() string {
	if c.Version == nil {
		return "*"
	}
	if c.Operator == "=" {
		return c.Version.String()
	}
	return c.Operator + c.Version.String()
}
