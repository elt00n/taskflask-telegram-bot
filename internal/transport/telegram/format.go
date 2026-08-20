package telegram

import (
	"fmt"
	"strings"
	"time"

	"github.com/elt00n/taskflask-telegram-bot/internal/domain"
)

const maxTasksPerMessage = 10

func formatCreatedTask(task domain.Task, location *time.Location) string {
	lines := []string{
		"✅ Дело создано",
		"",
		priorityIcon(task.Priority) + " " + task.Title,
		"ID: " + shortTaskID(task.ID),
	}
	lines = append(lines, formatTaskTimes(task, location)...)
	return strings.Join(lines, "\n")
}

func formatTaskList(tasks []domain.Task, location *time.Location) string {
	if len(tasks) == 0 {
		return "В этом списке пока нет задач."
	}

	limit := len(tasks)
	if limit > maxTasksPerMessage {
		limit = maxTasksPerMessage
	}
	lines := []string{"📋 Задачи", ""}
	for index, task := range tasks[:limit] {
		lines = append(lines, fmt.Sprintf(
			"%d. %s %s · %s",
			index+1,
			priorityIcon(task.Priority),
			task.Title,
			shortTaskID(task.ID),
		))
		for _, value := range formatTaskTimes(task, location) {
			lines = append(lines, "   "+value)
		}
	}
	if len(tasks) > limit {
		lines = append(lines, "", fmt.Sprintf("Ещё задач: %d", len(tasks)-limit))
	}

	return strings.Join(lines, "\n")
}

func formatUpdatedTask(message string, task domain.Task, location *time.Location) string {
	lines := []string{
		message,
		"",
		priorityIcon(task.Priority) + " " + task.Title,
		"ID: " + shortTaskID(task.ID),
	}
	lines = append(lines, formatTaskTimes(task, location)...)
	return strings.Join(lines, "\n")
}

func shortTaskID(taskID domain.TaskID) string {
	const visibleLength = 8
	value := string(taskID)
	if len(value) <= visibleLength {
		return value
	}
	return value[:visibleLength]
}

func formatTaskTimes(task domain.Task, location *time.Location) []string {
	lines := make([]string, 0, 2)
	if task.StartAt != nil {
		lines = append(lines, "🕒 "+task.StartAt.In(location).Format("02.01.2006 15:04"))
	}
	if task.Deadline != nil {
		lines = append(lines, "⏳ до "+task.Deadline.In(location).Format("02.01.2006 15:04"))
	}
	if task.StartAt == nil && task.Deadline == nil {
		lines = append(lines, "📥 Без срока")
	}
	return lines
}

func priorityIcon(priority domain.TaskPriority) string {
	switch priority {
	case domain.TaskPriorityCritical:
		return "🔴"
	case domain.TaskPriorityHigh:
		return "🟠"
	case domain.TaskPriorityLow:
		return "🟢"
	default:
		return "🟡"
	}
}
