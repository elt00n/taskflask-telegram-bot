package memory

import (
	"context"
	"sync"

	"github.com/elt00n/taskflask-telegram-bot/internal/domain"
	"github.com/elt00n/taskflask-telegram-bot/internal/repository"
)

var _ repository.ChatRepository = (*ChatRepository)(nil)

// ChatRepository хранит последнее известное состояние чатов в памяти.
type ChatRepository struct {
	mutex sync.Mutex
	chats map[domain.ChatID]domain.Chat
}

func NewChatRepository() *ChatRepository {
	return &ChatRepository{chats: make(map[domain.ChatID]domain.Chat)}
}

func (repo *ChatRepository) Upsert(ctx context.Context, chat domain.Chat) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	repo.mutex.Lock()
	defer repo.mutex.Unlock()

	if existing, found := repo.chats[chat.ID]; found {
		if existing.UpdatedAt.After(chat.UpdatedAt) {
			return nil
		}
		chat.CreatedAt = existing.CreatedAt
	}
	repo.chats[chat.ID] = chat
	return nil
}
