# todo

## requierd

- [yq](https://github.com/mikefarah/yq)
- [fzf](https://github.com/junegunn/fzf)

## usage

```shell
❯ go run main.go help
Usage: todo [command] [args...]
Commands:
  list    List all todos
  cd      Launch a subshell with cwd set to todoDir
  done    Mark a todo as done
  [todo_name] <project> Create a new todo with project
```

```shell
❯ go run main.go test_todo test_project

❯ cat $HOME/Documents/kurashidian/todo/test_todo.md 
---
title: test_todo
created_at: "2026-02-08"
project: test_project
---
{}
```
