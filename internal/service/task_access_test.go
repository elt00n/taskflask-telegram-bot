package service_test

import (
	"testing"
	"time"

	"github.com/elt00n/taskflask-telegram-bot/internal/domain"
	"github.com/elt00n/taskflask-telegram-bot/internal/service"
)

func TestTaskAccessPolicyCanView(t *testing.T) {
	task := testTask(t)
	policy := service.TaskAccessPolicy{}

	tests := []struct {
		name   string
		member domain.ChatMember
		want   bool
	}{
		{
			name:   "ordinary member of the task chat",
			member: activeMember(task.ChatID, 3),
			want:   true,
		},
		{
			name: "member of another chat",
			member: domain.ChatMember{
				ChatID: -200456,
				UserID: 3,
				Status: domain.ChatMemberStatusMember,
			},
			want: false,
		},
		{
			name: "user who left the chat",
			member: domain.ChatMember{
				ChatID: task.ChatID,
				UserID: 3,
				Status: domain.ChatMemberStatusLeft,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := policy.CanView(task, tt.member); got != tt.want {
				t.Errorf("CanView() = %t, want %t", got, tt.want)
			}
		})
	}
}

func TestTaskAccessPolicyCanEdit(t *testing.T) {
	task := testTask(t)
	policy := service.TaskAccessPolicy{}
	participants := []domain.TaskParticipant{
		{
			TaskID: task.ID,
			UserID: 2,
			Role:   domain.TaskParticipantRoleAssignee,
		},
		{
			TaskID: "another-task",
			UserID: 3,
			Role:   domain.TaskParticipantRoleEditor,
		},
	}

	tests := []struct {
		name   string
		member domain.ChatMember
		want   bool
	}{
		{name: "creator", member: activeMember(task.ChatID, task.CreatorID), want: true},
		{name: "assigned participant", member: activeMember(task.ChatID, 2), want: true},
		{name: "unassigned chat member", member: activeMember(task.ChatID, 3), want: false},
		{
			name: "assigned user from another chat",
			member: domain.ChatMember{
				ChatID: -200456,
				UserID: 2,
				Status: domain.ChatMemberStatusMember,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := policy.CanEdit(task, tt.member, participants); got != tt.want {
				t.Errorf("CanEdit() = %t, want %t", got, tt.want)
			}
		})
	}
}

func TestTaskAccessPolicyCanDelete(t *testing.T) {
	task := testTask(t)
	policy := service.TaskAccessPolicy{}

	if !policy.CanDelete(task, activeMember(task.ChatID, task.CreatorID)) {
		t.Error("creator must be allowed to delete the task")
	}
	if policy.CanDelete(task, activeMember(task.ChatID, 2)) {
		t.Error("assigned participant must not be allowed to delete the task")
	}
}

func testTask(t *testing.T) domain.Task {
	t.Helper()

	task, err := domain.NewTask(domain.NewTaskParams{
		ID:        "task-1",
		ChatID:    -100123,
		CreatorID: 1,
		Title:     "Подготовить отчёт",
	}, time.Date(2026, time.August, 20, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("NewTask() returned an unexpected error: %v", err)
	}

	return task
}

func activeMember(chatID domain.ChatID, userID domain.UserID) domain.ChatMember {
	return domain.ChatMember{
		ChatID: chatID,
		UserID: userID,
		Status: domain.ChatMemberStatusMember,
	}
}
