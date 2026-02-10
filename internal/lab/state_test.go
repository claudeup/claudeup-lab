package lab_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/claudeup/claudeup-lab/internal/lab"
)

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	store := lab.NewStateStore(dir)

	meta := &lab.Metadata{
		ID:          "abc-123",
		DisplayName: "myapp-base",
		Project:     "/home/user/code/myapp",
		ProjectName: "myapp",
		Profile:     "base",
		BareRepo:    "/home/user/.claudeup-lab/repos/myapp.git",
		Worktree:    "/home/user/.claudeup-lab/workspaces/myapp-base",
		Branch:      "lab/base",
		Created:     time.Date(2026, 2, 10, 12, 0, 0, 0, time.UTC),
	}

	if err := store.Save(meta); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := store.Load("abc-123")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if loaded.DisplayName != "myapp-base" {
		t.Errorf("DisplayName = %q, want %q", loaded.DisplayName, "myapp-base")
	}
	if loaded.Branch != "lab/base" {
		t.Errorf("Branch = %q, want %q", loaded.Branch, "lab/base")
	}
}

func TestList(t *testing.T) {
	dir := t.TempDir()
	store := lab.NewStateStore(dir)

	store.Save(&lab.Metadata{ID: "id-1", DisplayName: "lab-one"})
	store.Save(&lab.Metadata{ID: "id-2", DisplayName: "lab-two"})

	labs, err := store.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(labs) != 2 {
		t.Fatalf("List returned %d labs, want 2", len(labs))
	}
}

func TestDelete(t *testing.T) {
	dir := t.TempDir()
	store := lab.NewStateStore(dir)

	store.Save(&lab.Metadata{ID: "to-delete", DisplayName: "temp"})
	if err := store.Delete("to-delete"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, err := store.Load("to-delete")
	if err == nil {
		t.Error("Load after Delete should return error")
	}
}

func TestLoadNonexistent(t *testing.T) {
	dir := t.TempDir()
	store := lab.NewStateStore(dir)

	_, err := store.Load("does-not-exist")
	if err == nil {
		t.Error("Load nonexistent should return error")
	}
}

func TestListEmptyDir(t *testing.T) {
	dir := t.TempDir()
	store := lab.NewStateStore(dir)

	labs, err := store.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(labs) != 0 {
		t.Errorf("List returned %d labs, want 0", len(labs))
	}
}

func TestSaveCreatesDirectory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "state")
	store := lab.NewStateStore(dir)

	err := store.Save(&lab.Metadata{ID: "test", DisplayName: "test"})
	if err != nil {
		t.Fatalf("Save to nested dir: %v", err)
	}

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Error("Save should create state directory")
	}
}
