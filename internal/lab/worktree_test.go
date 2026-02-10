package lab_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/claudeup/claudeup-lab/internal/lab"
)

func initTestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	run(t, dir, "git", "init")
	run(t, dir, "git", "config", "user.email", "test@test.com")
	run(t, dir, "git", "config", "user.name", "Test")
	os.WriteFile(filepath.Join(dir, "README.md"), []byte("# test"), 0o644)
	run(t, dir, "git", "add", ".")
	run(t, dir, "git", "commit", "-m", "initial")
	return dir
}

func run(t *testing.T, dir string, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%s %v: %v\n%s", name, args, err, out)
	}
}

func TestEnsureBareRepo(t *testing.T) {
	source := initTestRepo(t)
	reposDir := filepath.Join(t.TempDir(), "repos")

	wt := lab.NewWorktreeManager(reposDir)
	barePath, err := wt.EnsureBareRepo(source, "testproject")
	if err != nil {
		t.Fatalf("EnsureBareRepo: %v", err)
	}

	if _, err := os.Stat(barePath); os.IsNotExist(err) {
		t.Error("bare repo should exist")
	}

	// Second call should refresh, not recreate
	barePath2, err := wt.EnsureBareRepo(source, "testproject")
	if err != nil {
		t.Fatalf("EnsureBareRepo (refresh): %v", err)
	}
	if barePath != barePath2 {
		t.Errorf("paths differ: %q vs %q", barePath, barePath2)
	}
}

func TestCreateWorktree(t *testing.T) {
	source := initTestRepo(t)
	reposDir := filepath.Join(t.TempDir(), "repos")
	wsDir := filepath.Join(t.TempDir(), "workspaces")

	wt := lab.NewWorktreeManager(reposDir)
	barePath, _ := wt.EnsureBareRepo(source, "testproject")

	wtPath := filepath.Join(wsDir, "test-lab")
	branch, err := wt.CreateWorktree(barePath, wtPath, "lab/test")
	if err != nil {
		t.Fatalf("CreateWorktree: %v", err)
	}

	if branch != "lab/test" {
		t.Errorf("branch = %q, want %q", branch, "lab/test")
	}

	readme := filepath.Join(wtPath, "README.md")
	if _, err := os.Stat(readme); os.IsNotExist(err) {
		t.Error("README.md should exist in worktree")
	}
}

func TestCreateWorktreeBranchCollision(t *testing.T) {
	source := initTestRepo(t)
	reposDir := filepath.Join(t.TempDir(), "repos")
	wsDir := filepath.Join(t.TempDir(), "workspaces")

	wt := lab.NewWorktreeManager(reposDir)
	barePath, _ := wt.EnsureBareRepo(source, "testproject")

	// Create first worktree on lab/test
	wt.CreateWorktree(barePath, filepath.Join(wsDir, "lab1"), "lab/test")

	// Second worktree on same branch should get a suffix
	branch, err := wt.CreateWorktree(barePath, filepath.Join(wsDir, "lab2"), "lab/test")
	if err != nil {
		t.Fatalf("CreateWorktree (collision): %v", err)
	}
	if branch == "lab/test" {
		t.Error("branch should have been renamed due to collision")
	}
	if len(branch) < 10 || branch[:9] != "lab/test-" {
		t.Errorf("branch = %q, want prefix %q", branch, "lab/test-")
	}
}

func TestRemoveWorktree(t *testing.T) {
	source := initTestRepo(t)
	reposDir := filepath.Join(t.TempDir(), "repos")
	wsDir := filepath.Join(t.TempDir(), "workspaces")

	wt := lab.NewWorktreeManager(reposDir)
	barePath, _ := wt.EnsureBareRepo(source, "testproject")

	wtPath := filepath.Join(wsDir, "test-lab")
	wt.CreateWorktree(barePath, wtPath, "lab/test")

	err := wt.RemoveWorktree(barePath, wtPath)
	if err != nil {
		t.Fatalf("RemoveWorktree: %v", err)
	}

	if _, err := os.Stat(wtPath); !os.IsNotExist(err) {
		t.Error("worktree directory should be removed")
	}
}

func TestWorktreeCount(t *testing.T) {
	source := initTestRepo(t)
	reposDir := filepath.Join(t.TempDir(), "repos")
	wsDir := filepath.Join(t.TempDir(), "workspaces")

	wt := lab.NewWorktreeManager(reposDir)
	barePath, _ := wt.EnsureBareRepo(source, "testproject")

	count, err := wt.WorktreeCount(barePath)
	if err != nil {
		t.Fatalf("WorktreeCount: %v", err)
	}
	// Bare repo lists itself as one worktree
	if count != 1 {
		t.Errorf("count = %d, want 1", count)
	}

	wt.CreateWorktree(barePath, filepath.Join(wsDir, "lab1"), "lab/test")
	count, _ = wt.WorktreeCount(barePath)
	if count != 2 {
		t.Errorf("count = %d, want 2", count)
	}
}
