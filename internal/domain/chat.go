package domain

import (
	"errors"
	"strings"
	"time"
)

// ChatType повторяет поддерживаемые приложением виды Telegram-чатов.
type ChatType string

const (
	ChatTypePrivate    ChatType = "private"
	ChatTypeGroup      ChatType = "group"
	ChatTypeSupergroup ChatType = "supergroup"
)

var (
	ErrUnsupportedChatType      = errors.New("unsupported chat type")
	ErrChatCreationTimeRequired = errors.New("chat creation time is required")
	ErrUnsupportedMemberStatus  = errors.New("unsupported chat member status")
	ErrMemberUpdateTimeRequired = errors.New("chat member update time is required")
)

// Chat представляет личный или групповой чат Telegram.
type Chat struct {
	ID        ChatID
	Title     string
	Type      ChatType
	CreatedAt time.Time
	UpdatedAt time.Time
}

// NewChat создаёт чат поддерживаемого типа.
func NewChat(id ChatID, title string, chatType ChatType, now time.Time) (Chat, error) {
	if !id.IsValid() {
		return Chat{}, ErrChatIDRequired
	}
	if !chatType.IsSupported() {
		return Chat{}, ErrUnsupportedChatType
	}
	if now.IsZero() {
		return Chat{}, ErrChatCreationTimeRequired
	}

	now = now.UTC()

	return Chat{
		ID:        id,
		Title:     strings.TrimSpace(title),
		Type:      chatType,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

// IsSupported сообщает, умеет ли приложение работать с таким типом чата.
func (chatType ChatType) IsSupported() bool {
	switch chatType {
	case ChatTypePrivate, ChatTypeGroup, ChatTypeSupergroup:
		return true
	default:
		return false
	}
}

// ChatMemberStatus описывает положение пользователя в чате.
type ChatMemberStatus string

const (
	ChatMemberStatusMember        ChatMemberStatus = "member"
	ChatMemberStatusAdministrator ChatMemberStatus = "administrator"
	ChatMemberStatusOwner         ChatMemberStatus = "owner"
	ChatMemberStatusLeft          ChatMemberStatus = "left"
	ChatMemberStatusBanned        ChatMemberStatus = "banned"
)

// ChatMember связывает пользователя с чатом и хранит актуальный статус участия.
type ChatMember struct {
	ChatID    ChatID
	UserID    UserID
	Status    ChatMemberStatus
	UpdatedAt time.Time
}

// NewChatMember создаёт или обновляет известную приложению связь с чатом.
func NewChatMember(chatID ChatID, userID UserID, status ChatMemberStatus, now time.Time) (ChatMember, error) {
	if !chatID.IsValid() {
		return ChatMember{}, ErrChatIDRequired
	}
	if !userID.IsValid() {
		return ChatMember{}, ErrUserIDRequired
	}
	if !status.IsSupported() {
		return ChatMember{}, ErrUnsupportedMemberStatus
	}
	if now.IsZero() {
		return ChatMember{}, ErrMemberUpdateTimeRequired
	}

	return ChatMember{
		ChatID:    chatID,
		UserID:    userID,
		Status:    status,
		UpdatedAt: now.UTC(),
	}, nil
}

// IsSupported сообщает, известен ли приложению такой статус Telegram.
func (status ChatMemberStatus) IsSupported() bool {
	switch status {
	case ChatMemberStatusMember,
		ChatMemberStatusAdministrator,
		ChatMemberStatusOwner,
		ChatMemberStatusLeft,
		ChatMemberStatusBanned:
		return true
	default:
		return false
	}
}

// IsActive сообщает, имеет ли пользователь доступ к текущим данным чата.
func (member ChatMember) IsActive() bool {
	switch member.Status {
	case ChatMemberStatusMember, ChatMemberStatusAdministrator, ChatMemberStatusOwner:
		return true
	default:
		return false
	}
}
