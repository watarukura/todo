package main

import (
	"fmt"
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
	todoDir := filepath.Join(os.Getenv("HOME"), "Documents/kurashidian/todo")

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
			// Note: cd in a child process won't affect the parent shell.
			// The original bash script had 'cd $todo_dir' which works because it's a function.
			// In Go, we just print the path or handle it.
			// But for a CLI tool, 'cd' usually isn't possible unless sourced.
			fmt.Println(todoDir)
			return
		case "done":
			markDone(todoDir, today)
			return
		default:
			// check if file exists
			filePath := filepath.Join(todoDir, cmd+".md")
			if _, err := os.Stat(filePath); err == nil {
				runEditor(editor, filePath)
				return
			}
			// create new todo
			createNewTodo(todoDir, cmd, "", today, editor)
			return
		}
	} else if len(args) == 2 {
		createNewTodo(todoDir, args[0], args[1], today, editor)
		return
	} else {
		// No args or > 2 args
		// bash: cd $todo_dir; printf -- "---\n{}\n---\n" | $EDOTOR; cd -
		// We'll just run editor with a temporary buffer or similar if possible,
		// but the bash script seems to just pipe into $EDITOR.
		// Most editors don't like piping like that without '-'
		runEditorWithPipe(editor, "---\n{}\n---\n")
		return
	}
}

func listTodos(todoDir string, editor string) {
	files, err := filepath.Glob(filepath.Join(todoDir, "*.md"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error listing files: %v\n", err)
		return
	}

	var allInputs []string
	for _, file := range files {
		info, err := extractFrontMatter(file)
		if err != nil {
			continue
		}
		filename := filepath.Base(file)
		allInputs = append(allInputs, fmt.Sprintf("%s\t%s\t%s\t%s", info.Title, info.CreatedAt, info.Project, filename))
	}

	if len(allInputs) == 0 {
		return
	}

	selected := runFzf(strings.Join(allInputs, "\n"))
	if selected != "" {
		parts := strings.Split(selected, "\t")
		if len(parts) >= 4 {
			runEditor(editor, filepath.Join(todoDir, parts[3]))
		}
	}
}

func markDone(todoDir string, today string) {
	files, err := filepath.Glob(filepath.Join(todoDir, "*.md"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error listing files: %v\n", err)
		return
	}

	var filenames []string
	for _, f := range files {
		filenames = append(filenames, filepath.Base(f))
	}

	doneFile := runFzf(strings.Join(filenames, "\n"))
	if doneFile == "" {
		return
	}

	fullPath := filepath.Join(todoDir, doneFile)

	// Update frontmatter
	content, err := os.ReadFile(fullPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
		return
	}

	parts := strings.SplitN(string(content), "---", 3)
	if len(parts) < 3 {
		fmt.Fprintf(os.Stderr, "Invalid frontmatter format in %s\n", doneFile)
		return
	}

	var fm FrontMatter
	err = yaml.Unmarshal([]byte(parts[1]), &fm)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing yaml: %v\n", err)
		return
	}

	fm.FinishedAt = today
	newYaml, _ := yaml.Marshal(fm)

	newContent := fmt.Sprintf("---\n%s---\n%s", string(newYaml), parts[2])
	err = os.WriteFile(fullPath, []byte(newContent), 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error writing file: %v\n", err)
		return
	}

	// Move to done/today/
	destDir := filepath.Join(todoDir, "done", today)
	err = os.MkdirAll(destDir, 0755)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating directory: %v\n", err)
		return
	}

	err = os.Rename(fullPath, filepath.Join(destDir, doneFile))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error moving file: %v\n", err)
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
		fmt.Fprintf(os.Stderr, "Error creating file: %v\n", err)
		return
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

func runFzf(input string) string {
	cmd := exec.Command("fzf")
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

func runEditorWithPipe(editor string, content string) {
	// Some editors might not support pipe. Bash: printf ... | $EDITOR
	// In Go we can try to pass it via stdin if the editor supports it (like vim -)
	// But the bash script just does '$EDITOR' without '-'.
	// Most editors will ignore stdin unless told otherwise.
	// If it's 'vi', it won't work without '-'.

	cmd := exec.Command(editor)
	if editor == "vi" || editor == "vim" || editor == "nvim" {
		cmd = exec.Command(editor, "-")
	}
	cmd.Stdin = strings.NewReader(content)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	_ = cmd.Run()
}
