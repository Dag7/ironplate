package envconfig

import (
	"testing"
	"time"
)

func TestGetString(t *testing.T) {
	t.Setenv("TEST_KEY", "hello")
	if got := GetString("TEST_KEY", "default"); got != "hello" {
		t.Errorf("GetString() = %q, want %q", got, "hello")
	}
	if got := GetString("MISSING_KEY", "default"); got != "default" {
		t.Errorf("GetString() = %q, want %q", got, "default")
	}
}

func TestMustGetString(t *testing.T) {
	t.Setenv("TEST_KEY", "value")
	if got := MustGetString("TEST_KEY"); got != "value" {
		t.Errorf("MustGetString() = %q, want %q", got, "value")
	}

	defer func() {
		if r := recover(); r == nil {
			t.Error("MustGetString() did not panic for missing key")
		}
	}()
	MustGetString("DEFINITELY_MISSING")
}

func TestGetInt(t *testing.T) {
	t.Setenv("PORT", "8080")
	if got := GetInt("PORT", 3000); got != 8080 {
		t.Errorf("GetInt() = %d, want %d", got, 8080)
	}
	if got := GetInt("MISSING", 3000); got != 3000 {
		t.Errorf("GetInt() = %d, want %d", got, 3000)
	}
	t.Setenv("BAD_INT", "abc")
	if got := GetInt("BAD_INT", 42); got != 42 {
		t.Errorf("GetInt() = %d, want %d", got, 42)
	}
}

func TestGetBool(t *testing.T) {
	tests := []struct {
		value    string
		expected bool
	}{
		{"true", true},
		{"TRUE", true},
		{"1", true},
		{"yes", true},
		{"on", true},
		{"false", false},
		{"0", false},
		{"no", false},
		{"off", false},
	}
	for _, tt := range tests {
		t.Setenv("BOOL_TEST", tt.value)
		if got := GetBool("BOOL_TEST", !tt.expected); got != tt.expected {
			t.Errorf("GetBool(%q) = %v, want %v", tt.value, got, tt.expected)
		}
	}
}

func TestGetDuration(t *testing.T) {
	t.Setenv("TIMEOUT", "5s")
	if got := GetDuration("TIMEOUT", time.Second); got != 5*time.Second {
		t.Errorf("GetDuration() = %v, want %v", got, 5*time.Second)
	}
	if got := GetDuration("MISSING", 10*time.Second); got != 10*time.Second {
		t.Errorf("GetDuration() = %v, want %v", got, 10*time.Second)
	}
}

func TestGetStringSlice(t *testing.T) {
	t.Setenv("HOSTS", "a,b,c")
	got := GetStringSlice("HOSTS", ",", nil)
	if len(got) != 3 || got[0] != "a" || got[1] != "b" || got[2] != "c" {
		t.Errorf("GetStringSlice() = %v, want [a b c]", got)
	}
}
