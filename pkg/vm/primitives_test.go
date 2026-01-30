package vm

import (
	"os"
	"testing"
	"time"

	"github.com/ytnobody/calcium-lang/pkg/value"
)

// =============================================================================
// Time/DateTime Primitive Tests
// =============================================================================

func TestPrimitiveTimeNow(t *testing.T) {
	before := time.Now().Unix()
	result := primitiveTimeNow()
	after := time.Now().Unix()

	if result.Type != value.TYPE_INT {
		t.Fatalf("expected int, got %s", result.Type)
	}

	ts := result.AsInt()
	if ts < before || ts > after {
		t.Errorf("timestamp %d not in range [%d, %d]", ts, before, after)
	}
}

func TestPrimitiveTimeNowMs(t *testing.T) {
	before := time.Now().UnixMilli()
	result := primitiveTimeNowMs()
	after := time.Now().UnixMilli()

	if result.Type != value.TYPE_INT {
		t.Fatalf("expected int, got %s", result.Type)
	}

	ts := result.AsInt()
	if ts < before || ts > after {
		t.Errorf("timestamp %d not in range [%d, %d]", ts, before, after)
	}
}

func TestPrimitiveTimeFormat(t *testing.T) {
	// 2024-01-15 10:30:00 UTC
	timestamp := int64(1705314600)

	tests := []struct {
		name     string
		args     []value.Value
		wantErr  bool
		expected string
	}{
		{
			name:     "basic format UTC",
			args:     []value.Value{value.Int(timestamp), value.String("2006-01-02"), value.String("UTC")},
			expected: "2024-01-15",
		},
		{
			name:     "time format UTC",
			args:     []value.Value{value.Int(timestamp), value.String("15:04:05"), value.String("UTC")},
			expected: "10:30:00",
		},
		{
			name:    "invalid timezone",
			args:    []value.Value{value.Int(timestamp), value.String("2006-01-02"), value.String("Invalid/Zone")},
			wantErr: true,
		},
		{
			name:    "wrong arg count",
			args:    []value.Value{value.Int(timestamp)},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := primitiveTimeFormat(tt.args...)
			if tt.wantErr {
				if result.Type != value.TYPE_FAILURE {
					t.Errorf("expected failure, got %v", result)
				}
				return
			}
			if result.Type != value.TYPE_SUCCESS {
				t.Fatalf("expected success, got %v", result)
			}
			val := result.AsSuccess()
			if val.AsString() != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, val.AsString())
			}
		})
	}
}

func TestPrimitiveTimeParse(t *testing.T) {
	tests := []struct {
		name     string
		args     []value.Value
		wantErr  bool
		expected int64
	}{
		{
			name:     "parse date UTC",
			args:     []value.Value{value.String("2024-01-15"), value.String("2006-01-02"), value.String("UTC")},
			expected: 1705276800, // 2024-01-15 00:00:00 UTC
		},
		{
			name:     "parse datetime UTC",
			args:     []value.Value{value.String("2024-01-15 10:30:00"), value.String("2006-01-02 15:04:05"), value.String("UTC")},
			expected: 1705314600,
		},
		{
			name:    "invalid format",
			args:    []value.Value{value.String("not-a-date"), value.String("2006-01-02"), value.String("UTC")},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := primitiveTimeParse(tt.args...)
			if tt.wantErr {
				if result.Type != value.TYPE_FAILURE {
					t.Errorf("expected failure, got %v", result)
				}
				return
			}
			if result.Type != value.TYPE_SUCCESS {
				t.Fatalf("expected success, got %v", result)
			}
			val := result.AsSuccess()
			if val.AsInt() != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, val.AsInt())
			}
		})
	}
}

func TestPrimitiveTimeComponents(t *testing.T) {
	// 2024-01-15 10:30:45 UTC (Monday)
	timestamp := int64(1705314645)

	result := primitiveTimeComponents(value.Int(timestamp), value.String("UTC"))
	if result.Type != value.TYPE_HASH {
		t.Fatalf("expected hash, got %s", result.Type)
	}

	hash := result.AsHash()

	tests := []struct {
		key      string
		expected int64
	}{
		{"year", 2024},
		{"month", 1},
		{"day", 15},
		{"hour", 10},
		{"minute", 30},
		{"second", 45},
		{"weekday", 1}, // Monday
	}

	for _, tt := range tests {
		val, ok := hash.Get(value.String(tt.key))
		if !ok {
			t.Errorf("missing key %q", tt.key)
			continue
		}
		if val.AsInt() != tt.expected {
			t.Errorf("%s: expected %d, got %d", tt.key, tt.expected, val.AsInt())
		}
	}
}

func TestPrimitiveTimeFromComponents(t *testing.T) {
	// Create 2024-01-15 10:30:45 UTC
	result := primitiveTimeFromComponents(
		value.Int(2024),
		value.Int(1),
		value.Int(15),
		value.Int(10),
		value.Int(30),
		value.Int(45),
		value.String("UTC"),
	)

	if result.Type != value.TYPE_INT {
		t.Fatalf("expected int, got %s", result.Type)
	}

	expected := int64(1705314645)
	if result.AsInt() != expected {
		t.Errorf("expected %d, got %d", expected, result.AsInt())
	}
}

func TestPrimitiveTimeAdd(t *testing.T) {
	tests := []struct {
		name     string
		ts       int64
		seconds  int64
		expected int64
	}{
		{"add positive", 1000, 500, 1500},
		{"add negative", 1000, -500, 500},
		{"add zero", 1000, 0, 1000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := primitiveTimeAdd(value.Int(tt.ts), value.Int(tt.seconds))
			if result.Type != value.TYPE_INT {
				t.Fatalf("expected int, got %s", result.Type)
			}
			if result.AsInt() != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, result.AsInt())
			}
		})
	}
}

func TestPrimitiveTimeDiff(t *testing.T) {
	result := primitiveTimeDiff(value.Int(1000), value.Int(400))
	if result.Type != value.TYPE_INT {
		t.Fatalf("expected int, got %s", result.Type)
	}
	if result.AsInt() != 600 {
		t.Errorf("expected 600, got %d", result.AsInt())
	}
}

// =============================================================================
// Environment Variable Primitive Tests
// =============================================================================

func TestPrimitiveEnvGetSet(t *testing.T) {
	testKey := "CALCIUM_TEST_ENV_VAR"
	testValue := "test_value_123"

	// Clean up after test
	defer os.Unsetenv(testKey)

	// Test get on non-existent var
	result := primitiveEnvGet(value.String(testKey))
	if result.Type != value.TYPE_FAILURE {
		t.Errorf("expected failure for non-existent var, got %v", result)
	}

	// Test set
	result = primitiveEnvSet(value.String(testKey), value.String(testValue))
	if result.Type != value.TYPE_SUCCESS {
		t.Fatalf("set failed: %v", result)
	}

	// Test get after set
	result = primitiveEnvGet(value.String(testKey))
	if result.Type != value.TYPE_SUCCESS {
		t.Fatalf("get failed: %v", result)
	}
	if result.AsSuccess().AsString() != testValue {
		t.Errorf("expected %q, got %q", testValue, result.AsSuccess().AsString())
	}
}

func TestPrimitiveEnvUnset(t *testing.T) {
	testKey := "CALCIUM_TEST_ENV_UNSET"
	os.Setenv(testKey, "temp_value")
	defer os.Unsetenv(testKey)

	// Verify it exists
	result := primitiveEnvGet(value.String(testKey))
	if result.Type != value.TYPE_SUCCESS {
		t.Fatal("var should exist before unset")
	}

	// Unset
	result = primitiveEnvUnset(value.String(testKey))
	if result.Type != value.TYPE_SUCCESS {
		t.Fatalf("unset failed: %v", result)
	}

	// Verify it's gone
	result = primitiveEnvGet(value.String(testKey))
	if result.Type != value.TYPE_FAILURE {
		t.Error("var should not exist after unset")
	}
}

func TestPrimitiveEnvList(t *testing.T) {
	testKey := "CALCIUM_TEST_ENV_LIST"
	testValue := "list_test_value"
	os.Setenv(testKey, testValue)
	defer os.Unsetenv(testKey)

	result := primitiveEnvList()
	if result.Type != value.TYPE_HASH {
		t.Fatalf("expected hash, got %s", result.Type)
	}

	hash := result.AsHash()
	val, ok := hash.Get(value.String(testKey))
	if !ok {
		t.Errorf("expected key %q in env list", testKey)
	} else if val.AsString() != testValue {
		t.Errorf("expected %q, got %q", testValue, val.AsString())
	}
}

func TestPrimitiveEnvErrors(t *testing.T) {
	// Test wrong arg types
	result := primitiveEnvGet(value.Int(123))
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for non-string arg")
	}

	result = primitiveEnvSet(value.String("key"), value.Int(123))
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for non-string value")
	}

	result = primitiveEnvList(value.String("extra"))
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for extra args")
	}
}
