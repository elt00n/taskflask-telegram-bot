package memory

import (
	"context"
	"sync"

	"github.com/elt00n/taskflask-telegram-bot/internal/domain"
	"github.com/elt00n/taskflask-telegram-bot/internal/repository"
)

var _ repository.ChatMemberRepository = (*ChatMemberRepository)(nil)

type chatMemberKey struct {
	chatID domain.ChatID
	userID domain.UserID
}

// ChatMemberRepository хранит последнее известное состояние участника чата.
type ChatMemberRepository struct {
	mutex   sync.RWMutex
	members map[chatMemberKey]domain.ChatMember
}

// NewChatMemberRepository возвращает пустое хранилище участников.
func NewChatMemberRepository() *ChatMemberRepository {
	return &ChatMemberRepository{
		members: make(map[chatMemberKey]domain.ChatMember),
	}
}

// Upsert добавляет участника или заменяет его состояние более новым.
func (repo *ChatMemberRepository) Upsert(
	ctx context.Context,
	member domain.ChatMember,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	repo.mutex.Lock()
	defer repo.mutex.Unlock()

	key := chatMemberKey{chatID: member.ChatID, userID: member.UserID}
	repo.members[key] = member

	return nil
}

// Get находит состояние конкретного пользователя в конкретном чате.
func (repo *ChatMemberRepository) Get(
	ctx context.Context,
	chatID domain.ChatID,
	userID domain.UserID,
) (domain.ChatMember, error) {
	if err := ctx.Err(); err != nil {
		return domain.ChatMember{}, err
	}

	repo.mutex.RLock()
	defer repo.mutex.RUnlock()

	member, exists := repo.members[chatMemberKey{chatID: chatID, userID: userID}]
	if !exists {
		return domain.ChatMember{}, repository.ErrChatMemberNotFound
	}

	return member, nil
}
