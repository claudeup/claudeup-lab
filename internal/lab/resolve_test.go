package lab_test

import (
	"testing"

	"github.com/claudeup/claudeup-lab/internal/lab"
)

func setupResolver(t *testing.T) (*lab.Resolver, *lab.StateStore) {
	t.Helper()
	dir := t.TempDir()
	store := lab.NewStateStore(dir)

	store.Save(&lab.Metadata{
		ID:          "abc-def-123",
		DisplayName: "myapp-base",
		ProjectName: "myapp",
		Profile:     "base",
		Worktree:    "/tmp/workspaces/myapp-base",
	})
	store.Save(&lab.Metadata{
		ID:          "xyz-789-456",
		DisplayName: "other-experimental",
		ProjectName: "other",
		Profile:     "experimental",
		Worktree:    "/tmp/workspaces/other-experimental",
	})

	return lab.NewResolver(store), store
}

func TestResolveExactUUID(t *testing.T) {
	r, _ := setupResolver(t)
	meta, err := r.Resolve("abc-def-123")
	if err != nil {
		t.Fatalf("Resolve exact UUID: %v", err)
	}
	if meta.DisplayName != "myapp-base" {
		t.Errorf("got %q, want %q", meta.DisplayName, "myapp-base")
	}
}

func TestResolveDisplayName(t *testing.T) {
	r, _ := setupResolver(t)
	meta, err := r.Resolve("myapp-base")
	if err != nil {
		t.Fatalf("Resolve display name: %v", err)
	}
	if meta.ID != "abc-def-123" {
		t.Errorf("got %q, want %q", meta.ID, "abc-def-123")
	}
}

func TestResolvePartialUUID(t *testing.T) {
	r, _ := setupResolver(t)
	meta, err := r.Resolve("abc-def")
	if err != nil {
		t.Fatalf("Resolve partial UUID: %v", err)
	}
	if meta.ID != "abc-def-123" {
		t.Errorf("got %q, want %q", meta.ID, "abc-def-123")
	}
}

func TestResolveProjectName(t *testing.T) {
	r, _ := setupResolver(t)
	meta, err := r.Resolve("myapp")
	if err != nil {
		t.Fatalf("Resolve project name: %v", err)
	}
	if meta.ID != "abc-def-123" {
		t.Errorf("got %q, want %q", meta.ID, "abc-def-123")
	}
}

func TestResolveProfileName(t *testing.T) {
	r, _ := setupResolver(t)
	meta, err := r.Resolve("experimental")
	if err != nil {
		t.Fatalf("Resolve profile name: %v", err)
	}
	if meta.ID != "xyz-789-456" {
		t.Errorf("got %q, want %q", meta.ID, "xyz-789-456")
	}
}

func TestResolveAmbiguous(t *testing.T) {
	dir := t.TempDir()
	store := lab.NewStateStore(dir)
	store.Save(&lab.Metadata{ID: "id-1", DisplayName: "a", ProjectName: "shared", Profile: "p1"})
	store.Save(&lab.Metadata{ID: "id-2", DisplayName: "b", ProjectName: "shared", Profile: "p2"})

	r := lab.NewResolver(store)
	_, err := r.Resolve("shared")
	if err == nil {
		t.Error("expected ambiguous error")
	}
}

func TestResolveNoMatch(t *testing.T) {
	r, _ := setupResolver(t)
	_, err := r.Resolve("nonexistent")
	if err == nil {
		t.Error("expected no match error")
	}
}

func TestResolveByCWD(t *testing.T) {
	r, _ := setupResolver(t)
	meta, err := r.ResolveByCWD("/tmp/workspaces/myapp-base/subdir")
	if err != nil {
		t.Fatalf("ResolveByCWD: %v", err)
	}
	if meta.ID != "abc-def-123" {
		t.Errorf("got %q, want %q", meta.ID, "abc-def-123")
	}
}

func TestResolveByCWDNoMatch(t *testing.T) {
	r, _ := setupResolver(t)
	_, err := r.ResolveByCWD("/some/other/path")
	if err == nil {
		t.Error("expected no match error for unrelated cwd")
	}
}
