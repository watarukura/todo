package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExtractFrontMatterOK(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "todo.md")
	content := strings.Join([]string{
		"---",
		"title: test",
		"created_at: 2025-01-01",
		"project: demo",
		"---",
		"body",
		"",
	}, "\n")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	fm, err := extractFrontMatter(path)
	if err != nil {
		t.Fatalf("extractFrontMatter error: %v", err)
	}
	if fm.Title != "test" {
		t.Fatalf("unexpected title: %q", fm.Title)
	}
	if fm.CreatedAt != "2025-01-01" {
		t.Fatalf("unexpected created_at: %q", fm.CreatedAt)
	}
	if fm.Project != "demo" {
		t.Fatalf("unexpected project: %q", fm.Project)
	}
}

func TestExtractFrontMatterMissing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "no_frontmatter.md")
	if err := os.WriteFile(path, []byte("no frontmatter"), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	_, err := extractFrontMatter(path)
	if err == nil {
		t.Fatal("expected error for missing frontmatter")
	}
}

func TestCreateNewTodoWritesFile(t *testing.T) {
	dir := t.TempDir()
	title := "a/b\x00c"
	project := "proj"
	today := "2025-01-01"

	createNewTodo(dir, title, project, today, "/usr/bin/true")

	path := filepath.Join(dir, "a_bc.md")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected file to exist: %v", err)
	}

	fm, err := extractFrontMatter(path)
	if err != nil {
		t.Fatalf("extractFrontMatter error: %v", err)
	}
	if fm.Title != title {
		t.Fatalf("unexpected title: %q", fm.Title)
	}
	if fm.CreatedAt != today {
		t.Fatalf("unexpected created_at: %q", fm.CreatedAt)
	}
	if fm.Project != project {
		t.Fatalf("unexpected project: %q", fm.Project)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	if !strings.Contains(string(content), "\n{}\n") {
		t.Fatalf("expected empty body template")
	}
}
