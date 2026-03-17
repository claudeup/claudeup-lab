package commands

import (
	"strings"
	"testing"
)

func TestParseEnvFlags(t *testing.T) {
	t.Run("valid key=value pairs", func(t *testing.T) {
		got, err := parseEnvFlags([]string{"FOO=bar", "BAZ=qux"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got["FOO"] != "bar" || got["BAZ"] != "qux" {
			t.Errorf("unexpected result: %v", got)
		}
	})

	t.Run("value can contain equals", func(t *testing.T) {
		got, err := parseEnvFlags([]string{"CONN=host=localhost;port=5432"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got["CONN"] != "host=localhost;port=5432" {
			t.Errorf("got %q, want %q", got["CONN"], "host=localhost;port=5432")
		}
	})

	t.Run("empty value is valid", func(t *testing.T) {
		got, err := parseEnvFlags([]string{"FOO="})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got["FOO"] != "" {
			t.Errorf("got %q, want empty string", got["FOO"])
		}
	})

	t.Run("last flag wins for duplicate keys", func(t *testing.T) {
		got, err := parseEnvFlags([]string{"FOO=first", "FOO=second"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got["FOO"] != "second" {
			t.Errorf("got %q, want second", got["FOO"])
		}
	})

	t.Run("errors on missing equals", func(t *testing.T) {
		_, err := parseEnvFlags([]string{"INVALID"})
		if err == nil {
			t.Fatal("expected error for missing =")
		}
		if !strings.Contains(err.Error(), "expected KEY=VALUE") {
			t.Errorf("error %q should contain 'expected KEY=VALUE'", err.Error())
		}
	})

	t.Run("errors on empty key", func(t *testing.T) {
		_, err := parseEnvFlags([]string{"=value"})
		if err == nil {
			t.Fatal("expected error for empty key")
		}
		if !strings.Contains(err.Error(), "empty key") {
			t.Errorf("error %q should contain 'empty key'", err.Error())
		}
	})

	t.Run("nil input returns nil", func(t *testing.T) {
		got, err := parseEnvFlags(nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != nil {
			t.Errorf("expected nil, got %v", got)
		}
	})
}
