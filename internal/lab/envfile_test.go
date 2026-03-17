// internal/lab/envfile_test.go
package lab

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseEnvFile(t *testing.T) {
	t.Run("parses valid key=value pairs", func(t *testing.T) {
		path := writeEnvFile(t, "FOO=bar\nBAZ=qux\n")
		got, err := ParseEnvFile(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := map[string]string{"FOO": "bar", "BAZ": "qux"}
		assertEnvEqual(t, got, want)
	})

	t.Run("skips comments and blank lines", func(t *testing.T) {
		path := writeEnvFile(t, "# comment\nFOO=bar\n\n  \n# another\nBAZ=qux\n")
		got, err := ParseEnvFile(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := map[string]string{"FOO": "bar", "BAZ": "qux"}
		assertEnvEqual(t, got, want)
	})

	t.Run("value can contain equals signs", func(t *testing.T) {
		path := writeEnvFile(t, "CONN=host=localhost;port=5432\n")
		got, err := ParseEnvFile(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got["CONN"] != "host=localhost;port=5432" {
			t.Errorf("got %q, want %q", got["CONN"], "host=localhost;port=5432")
		}
	})

	t.Run("quotes are literal", func(t *testing.T) {
		path := writeEnvFile(t, `FOO="bar baz"`+"\n")
		got, err := ParseEnvFile(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got["FOO"] != `"bar baz"` {
			t.Errorf("got %q, want %q", got["FOO"], `"bar baz"`)
		}
	})

	t.Run("empty value is valid", func(t *testing.T) {
		path := writeEnvFile(t, "FOO=\n")
		got, err := ParseEnvFile(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got["FOO"] != "" {
			t.Errorf("got %q, want empty string", got["FOO"])
		}
	})

	t.Run("errors on line without equals", func(t *testing.T) {
		path := writeEnvFile(t, "FOO=bar\nINVALID\n")
		_, err := ParseEnvFile(path)
		if err == nil {
			t.Fatal("expected error for line without =")
		}
		wantMsg := "expected KEY=VALUE"
		if !strings.Contains(err.Error(), wantMsg) {
			t.Errorf("error %q should contain %q", err.Error(), wantMsg)
		}
		if !strings.Contains(err.Error(), "line 2") {
			t.Errorf("error %q should reference line 2", err.Error())
		}
	})

	t.Run("errors on empty key", func(t *testing.T) {
		path := writeEnvFile(t, "=value\n")
		_, err := ParseEnvFile(path)
		if err == nil {
			t.Fatal("expected error for empty key")
		}
		if !strings.Contains(err.Error(), "empty key") {
			t.Errorf("error %q should contain 'empty key'", err.Error())
		}
	})

	t.Run("errors on missing file", func(t *testing.T) {
		_, err := ParseEnvFile("/nonexistent/path/env.txt")
		if err == nil {
			t.Fatal("expected error for missing file")
		}
	})

	t.Run("last value wins for duplicate keys", func(t *testing.T) {
		path := writeEnvFile(t, "FOO=first\nFOO=second\n")
		got, err := ParseEnvFile(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got["FOO"] != "second" {
			t.Errorf("got %q, want %q", got["FOO"], "second")
		}
	})
}

func writeEnvFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "env")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write env file: %v", err)
	}
	return path
}

func assertEnvEqual(t *testing.T, got, want map[string]string) {
	t.Helper()
	if len(got) != len(want) {
		t.Errorf("got %d keys, want %d", len(got), len(want))
	}
	for k, wv := range want {
		if gv, ok := got[k]; !ok {
			t.Errorf("missing key %q", k)
		} else if gv != wv {
			t.Errorf("key %q: got %q, want %q", k, gv, wv)
		}
	}
}
