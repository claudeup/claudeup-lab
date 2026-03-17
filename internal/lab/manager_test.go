package lab

import (
	"testing"
)

func TestMergeEnvVars(t *testing.T) {
	t.Run("file only", func(t *testing.T) {
		path := writeEnvFile(t, "FOO=from-file\nBAR=also-file\n")
		got, err := mergeEnvVars(path, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got["FOO"] != "from-file" || got["BAR"] != "also-file" {
			t.Errorf("unexpected result: %v", got)
		}
	})

	t.Run("flags only", func(t *testing.T) {
		got, err := mergeEnvVars("", map[string]string{"FOO": "from-flag"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got["FOO"] != "from-flag" {
			t.Errorf("got %q, want from-flag", got["FOO"])
		}
	})

	t.Run("flags override file", func(t *testing.T) {
		path := writeEnvFile(t, "FOO=from-file\nBAR=file-only\n")
		flags := map[string]string{"FOO": "from-flag"}
		got, err := mergeEnvVars(path, flags)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got["FOO"] != "from-flag" {
			t.Errorf("FOO = %q, want from-flag", got["FOO"])
		}
		if got["BAR"] != "file-only" {
			t.Errorf("BAR = %q, want file-only", got["BAR"])
		}
	})

	t.Run("both empty returns empty map", func(t *testing.T) {
		got, err := mergeEnvVars("", nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 0 {
			t.Errorf("expected empty map, got %v", got)
		}
	})

	t.Run("invalid env file returns error", func(t *testing.T) {
		path := writeEnvFile(t, "INVALID\n")
		_, err := mergeEnvVars(path, nil)
		if err == nil {
			t.Fatal("expected error for invalid env file")
		}
	})
}
