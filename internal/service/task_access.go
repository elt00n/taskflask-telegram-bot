// Package service содержит сценарии приложения и правила доступа.
package service

import "github.com/elt00n/taskflask-telegram-bot/internal/domain"

// TaskAccessPolicy проверяет права пользователя на операции с задачей.
// Структура пока не хранит состояние, но отдельный тип делает правила явными.
type TaskAccessPolicy struct{}

// CanView разрешает просмотр любому активному участнику того же чата.
func (TaskAccessPolicy) CanView(task domain.Task, member domain.ChatMember) bool {
	return member.ChatID == task.ChatID && member.IsActive()
}

// CanEdit разрешает изменение создателю или назначенному редактору задачи.
func (policy TaskAccessPolicy) CanEdit(
	task domain.Task,
	member domain.ChatMember,
	participants []domain.TaskParticipant,
) bool {
	if !policy.CanView(task, member) {
		return false
	}
	if member.UserID == task.CreatorID {
		return true
	}

	for _, participant := range participants {
		if participant.TaskID == task.ID &&
			participant.UserID == member.UserID &&
			participant.CanEdit() {
			return true
		}
	}

	return false
}

// CanDelete разрешает удаление только создателю задачи, который остаётся в чате.
func (policy TaskAccessPolicy) CanDelete(task domain.Task, member domain.ChatMember) bool {
	return policy.CanView(task, member) && member.UserID == task.CreatorID
}
