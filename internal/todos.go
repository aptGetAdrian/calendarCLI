package calendar

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

type Todo struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

const todosFilePath = "todos.json"

func LoadTodos() ([]Todo, error) {
	data, err := os.ReadFile(todosFilePath)
	if os.IsNotExist(err) {
		return []Todo{}, nil
	}
	if err != nil {
		return nil, err
	}
	var todos []Todo
	if err := json.Unmarshal(data, &todos); err != nil {
		return nil, err
	}
	return todos, nil
}

func saveTodos(todos []Todo) error {
	data, err := json.MarshalIndent(todos, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(todosFilePath, data, 0644)
}

func AddTodo(title, description string) ([]Todo, error) {
	todos, err := LoadTodos()
	if err != nil {
		return nil, err
	}
	todo := Todo{
		ID:          fmt.Sprintf("%d", time.Now().UnixNano()),
		Title:       title,
		Description: description,
	}
	todos = append(todos, todo)
	return todos, saveTodos(todos)
}

func UpdateTodo(id, title, description string) ([]Todo, error) {
	todos, err := LoadTodos()
	if err != nil {
		return nil, err
	}
	for i, t := range todos {
		if t.ID == id {
			todos[i].Title = title
			todos[i].Description = description
			break
		}
	}
	return todos, saveTodos(todos)
}

func DeleteTodo(id string) ([]Todo, error) {
	todos, err := LoadTodos()
	if err != nil {
		return nil, err
	}
	filtered := make([]Todo, 0, len(todos))
	for _, t := range todos {
		if t.ID != id {
			filtered = append(filtered, t)
		}
	}
	return filtered, saveTodos(filtered)
}
