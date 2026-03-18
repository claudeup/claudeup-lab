package commands

import (
	"os"
	"path/filepath"
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

func TestResolveInitScript(t *testing.T) {
	t.Run("empty path is a no-op", func(t *testing.T) {
		got, err := resolveInitScript("")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "" {
			t.Errorf("got %q, want empty string", got)
		}
	})

	t.Run("resolves existing file to absolute path", func(t *testing.T) {
		tmp := t.TempDir()
		script := filepath.Join(tmp, "setup.sh")
		if err := os.WriteFile(script, []byte("#!/bin/bash\n"), 0o755); err != nil {
			t.Fatal(err)
		}

		got, err := resolveInitScript(script)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !filepath.IsAbs(got) {
			t.Errorf("expected absolute path, got %q", got)
		}
		if got != script {
			t.Errorf("got %q, want %q", got, script)
		}
	})

	t.Run("resolves relative path to absolute", func(t *testing.T) {
		tmp := t.TempDir()
		script := filepath.Join(tmp, "setup.sh")
		if err := os.WriteFile(script, []byte("#!/bin/bash\n"), 0o755); err != nil {
			t.Fatal(err)
		}

		// Get path relative to cwd
		cwd, err := os.Getwd()
		if err != nil {
			t.Fatal(err)
		}
		rel, err := filepath.Rel(cwd, script)
		if err != nil {
			t.Fatal(err)
		}

		got, err := resolveInitScript(rel)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !filepath.IsAbs(got) {
			t.Errorf("expected absolute path, got %q", got)
		}
		if got != script {
			t.Errorf("got %q, want %q", got, script)
		}
	})

	t.Run("errors on non-existent file", func(t *testing.T) {
		_, err := resolveInitScript("/no/such/file.sh")
		if err == nil {
			t.Fatal("expected error for non-existent file")
		}
		if !strings.Contains(err.Error(), "init script not found") {
			t.Errorf("error %q should contain 'init script not found'", err.Error())
		}
	})

	t.Run("errors on directory path", func(t *testing.T) {
		tmp := t.TempDir()
		_, err := resolveInitScript(tmp)
		if err == nil {
			t.Fatal("expected error for directory path")
		}
		if !strings.Contains(err.Error(), "not a regular file") {
			t.Errorf("error %q should contain 'not a regular file'", err.Error())
		}
	})

	t.Run("wraps non-existence errors distinctly from other stat errors", func(t *testing.T) {
		// Non-existent path should say "not found"
		_, err := resolveInitScript("/no/such/file.sh")
		if err == nil {
			t.Fatal("expected error")
		}
		if !strings.Contains(err.Error(), "not found") {
			t.Errorf("error %q should contain 'not found'", err.Error())
		}
		if strings.Contains(err.Error(), "not accessible") {
			t.Errorf("non-existent file error should say 'not found', not 'not accessible'")
		}
	})
}
