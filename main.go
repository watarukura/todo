package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type FrontMatter struct {
	Title      string `yaml:"title"`
	CreatedAt  string `yaml:"created_at"`
	Project    string `yaml:"project,omitempty"`
	FinishedAt string `yaml:"finished_at,omitempty"`
}

func main() {
	todoDir := defaultTodoDir()

	today := time.Now().Format("2006-01-02")
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}

	args := os.Args[1:]

	if len(args) == 1 {
		cmd := args[0]
		switch cmd {
		case "list":
			listTodos(todoDir, editor)
			return
		case "cd":
			// Launch a subshell with cwd set to todoDir (chezmoi-like behavior).
			runSubshell(todoDir)
			return
		case "done":
			markDone(todoDir, today)
			return
		case "help":
		case "-h":
		case "-help":
		case "--help":
			help()
			return
		default:
			// ファイルが存在するか確認
			filePath := filepath.Join(todoDir, cmd+".md")
			if _, err := os.Stat(filePath); err == nil {
				runEditor(editor, filePath)
				return
			}
			createNewTodo(todoDir, cmd, "", today, editor)
			return
		}
	} else if len(args) == 2 {
		createNewTodo(todoDir, args[0], args[1], today, editor)
		return
	}

	// 引数なし、または2つより多い場合はhelp
	help()
}

func defaultTodoDir() string {
	if envDir, ok := os.LookupEnv("TODO_DIR"); ok && envDir != "" {
		if strings.HasPrefix(envDir, "~") {
			home := os.Getenv("HOME")
			if home != "" {
				return filepath.Clean(filepath.Join(home, strings.TrimPrefix(envDir, "~")))
			}
		}
		return filepath.Clean(envDir)
	}
	return filepath.Join(os.Getenv("HOME"), "Documents/todo")
}

func help() {
	fmt.Println("Usage: todo [command] [args...]")
	fmt.Println("Commands:")
	fmt.Println("  list    List all todos")
	fmt.Println("  cd      Launch a subshell with cwd set to todoDir")
	fmt.Println("  done    Mark a todo as done")
	fmt.Println("  [todo_name] <project> Create a new todo with project")
	fmt.Println("Env:")
	fmt.Println("  TODO_DIR    Override the todo directory (default: ~/Documents/todo)")
}

func listTodos(todoDir string, editor string) {
	files, err := filepath.Glob(filepath.Join(todoDir, "*.md"))
	if err != nil {
		log.Fatalf("Error listing files: %v\n", err)
	}

	if len(files) == 0 {
		return
	}

	var filenames []string
	for _, f := range files {
		filenames = append(filenames, filepath.Base(f))
	}

	selected := runFzf(strings.Join(filenames, "\n"), todoDir)
	if selected != "" {
		runEditor(editor, filepath.Join(todoDir, selected))
	}
}

func markDone(todoDir string, today string) {
	files, err := filepath.Glob(filepath.Join(todoDir, "*.md"))
	if err != nil {
		log.Fatalf("Error listing files: %v\n", err)
	}

	var filenames []string
	for _, f := range files {
		filenames = append(filenames, filepath.Base(f))
	}

	doneFile := runFzf(strings.Join(filenames, "\n"), todoDir)
	if doneFile == "" {
		return
	}

	fullPath := filepath.Join(todoDir, doneFile)

	// フロントマターを更新
	content, err := os.ReadFile(fullPath)
	if err != nil {
		log.Fatalf("Error reading file: %v\n", err)
	}

	parts := strings.SplitN(string(content), "---", 3)
	if len(parts) < 3 {
		log.Fatalf("Invalid frontmatter format in %s\n", doneFile)
	}

	var fm FrontMatter
	err = yaml.Unmarshal([]byte(parts[1]), &fm)
	if err != nil {
		log.Fatalf("Error parsing yaml: %v\n", err)
	}

	fm.FinishedAt = today
	newYaml, _ := yaml.Marshal(fm)

	newContent := fmt.Sprintf("---\n%s---\n%s", string(newYaml), parts[2])
	err = os.WriteFile(fullPath, []byte(newContent), 0644)
	if err != nil {
		log.Fatalf("Error writing file: %v\n", err)
	}

	// done/today/ に移動
	destDir := filepath.Join(todoDir, "done", today)
	err = os.MkdirAll(destDir, 0755)
	if err != nil {
		log.Fatalf("Error creating directory: %v\n", err)
	}

	err = os.Rename(fullPath, filepath.Join(destDir, doneFile))
	if err != nil {
		log.Fatalf("Error moving file: %v\n", err)
	}
}

func createNewTodo(todoDir, title, project, today, editor string) {
	escapedFilename := strings.ReplaceAll(title, "/", "_")
	escapedFilename = strings.ReplaceAll(escapedFilename, "\x00", "")

	filePath := filepath.Join(todoDir, escapedFilename+".md")

	fm := FrontMatter{
		Title:     title,
		CreatedAt: today,
		Project:   project,
	}

	fmYaml, _ := yaml.Marshal(fm)
	content := fmt.Sprintf("---\n%s---\n{}\n", string(fmYaml))

	err := os.WriteFile(filePath, []byte(content), 0644)
	if err != nil {
		log.Fatalf("Error creating file: %v\n", err)
	}

	runEditor(editor, filePath)
}

func extractFrontMatter(path string) (FrontMatter, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return FrontMatter{}, err
	}

	parts := strings.SplitN(string(content), "---", 3)
	if len(parts) < 3 {
		return FrontMatter{}, fmt.Errorf("no frontmatter")
	}

	var fm FrontMatter
	err = yaml.Unmarshal([]byte(parts[1]), &fm)
	return fm, err
}

func runFzf(input string, todoDir string) string {
	cmd := exec.Command("fzf", "--preview", "sed -n '1,200p' {}")
	cmd.Dir = todoDir
	cmd.Stdin = strings.NewReader(input)
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func runEditor(editor string, path string) {
	cmd := exec.Command(editor, path)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	_ = cmd.Run()
}

func runSubshell(dir string) {
	// Spawn the user's shell as a subshell in the target directory.
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "sh"
	}
	cmd := exec.Command(shell)
	cmd.Dir = dir
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	_ = cmd.Run()
}
