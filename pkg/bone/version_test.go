package bone

import "testing"

func TestParseVersion(t *testing.T) {
	tests := []struct {
		input    string
		expected Version
		wantErr  bool
	}{
		{"1.0.0", Version{1, 0, 0}, false},
		{"2.3.4", Version{2, 3, 4}, false},
		{"v1.2.3", Version{1, 2, 3}, false},
		{"0.0.1", Version{0, 0, 1}, false},
		{"10.20.30", Version{10, 20, 30}, false},
		{"1.0.0-beta", Version{1, 0, 0}, false},
		{"invalid", Version{}, true},
	}

	for _, tt := range tests {
		v, err := ParseVersion(tt.input)
		if tt.wantErr {
			if err == nil {
				t.Errorf("ParseVersion(%q) expected error", tt.input)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParseVersion(%q) error: %v", tt.input, err)
			continue
		}
		if *v != tt.expected {
			t.Errorf("ParseVersion(%q) = %v, want %v", tt.input, *v, tt.expected)
		}
	}
}

func TestVersionCompare(t *testing.T) {
	tests := []struct {
		v1       string
		v2       string
		expected int
	}{
		{"1.0.0", "1.0.0", 0},
		{"1.0.0", "2.0.0", -1},
		{"2.0.0", "1.0.0", 1},
		{"1.1.0", "1.0.0", 1},
		{"1.0.1", "1.0.0", 1},
		{"1.0.0", "1.1.0", -1},
		{"1.0.0", "1.0.1", -1},
	}

	for _, tt := range tests {
		v1, _ := ParseVersion(tt.v1)
		v2, _ := ParseVersion(tt.v2)
		result := v1.Compare(v2)
		if result != tt.expected {
			t.Errorf("%s.Compare(%s) = %d, want %d", tt.v1, tt.v2, result, tt.expected)
		}
	}
}

func TestParseConstraint(t *testing.T) {
	tests := []struct {
		input    string
		operator string
		version  string
	}{
		{"1.0.0", "=", "1.0.0"},
		{"=1.0.0", "=", "1.0.0"},
		{"^1.0.0", "^", "1.0.0"},
		{"~1.0.0", "~", "1.0.0"},
		{">=1.0.0", ">=", "1.0.0"},
		{"<=1.0.0", "<=", "1.0.0"},
		{">1.0.0", ">", "1.0.0"},
		{"<1.0.0", "<", "1.0.0"},
		{"*", "*", ""},
		{"latest", "*", ""},
	}

	for _, tt := range tests {
		c, err := ParseConstraint(tt.input)
		if err != nil {
			t.Errorf("ParseConstraint(%q) error: %v", tt.input, err)
			continue
		}
		if c.Operator != tt.operator {
			t.Errorf("ParseConstraint(%q).Operator = %q, want %q", tt.input, c.Operator, tt.operator)
		}
		if tt.version != "" {
			if c.Version == nil {
				t.Errorf("ParseConstraint(%q).Version is nil", tt.input)
			} else if c.Version.String() != tt.version {
				t.Errorf("ParseConstraint(%q).Version = %s, want %s", tt.input, c.Version.String(), tt.version)
			}
		}
	}
}

func TestConstraintMatches(t *testing.T) {
	tests := []struct {
		constraint string
		version    string
		expected   bool
	}{
		// Exact match
		{"1.0.0", "1.0.0", true},
		{"1.0.0", "1.0.1", false},
		{"=1.0.0", "1.0.0", true},

		// Greater than
		{">1.0.0", "1.0.1", true},
		{">1.0.0", "1.0.0", false},
		{">1.0.0", "0.9.0", false},

		// Greater than or equal
		{">=1.0.0", "1.0.0", true},
		{">=1.0.0", "1.0.1", true},
		{">=1.0.0", "0.9.0", false},

		// Less than
		{"<2.0.0", "1.0.0", true},
		{"<2.0.0", "2.0.0", false},
		{"<2.0.0", "2.0.1", false},

		// Less than or equal
		{"<=2.0.0", "2.0.0", true},
		{"<=2.0.0", "1.9.0", true},
		{"<=2.0.0", "2.0.1", false},

		// Caret (compatible with)
		{"^1.0.0", "1.0.0", true},
		{"^1.0.0", "1.9.9", true},
		{"^1.0.0", "2.0.0", false},
		{"^1.2.3", "1.2.3", true},
		{"^1.2.3", "1.3.0", true},
		{"^1.2.3", "1.2.2", false},
		{"^0.2.3", "0.2.5", true},
		{"^0.2.3", "0.3.0", false},

		// Tilde (approximately)
		{"~1.2.3", "1.2.3", true},
		{"~1.2.3", "1.2.9", true},
		{"~1.2.3", "1.3.0", false},
		{"~1.2.3", "1.2.2", false},

		// Any
		{"*", "1.0.0", true},
		{"*", "99.99.99", true},
		{"latest", "1.0.0", true},
	}

	for _, tt := range tests {
		c, err := ParseConstraint(tt.constraint)
		if err != nil {
			t.Errorf("ParseConstraint(%q) error: %v", tt.constraint, err)
			continue
		}
		v, err := ParseVersion(tt.version)
		if err != nil {
			t.Errorf("ParseVersion(%q) error: %v", tt.version, err)
			continue
		}
		result := c.Matches(v)
		if result != tt.expected {
			t.Errorf("Constraint(%q).Matches(%q) = %v, want %v", tt.constraint, tt.version, result, tt.expected)
		}
	}
}
