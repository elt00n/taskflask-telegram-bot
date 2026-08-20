package memory

import (
	"context"
	"errors"
	"strings"
	"sync"

	"github.com/elt00n/taskflask-telegram-bot/internal/domain"
	"github.com/elt00n/taskflask-telegram-bot/internal/repository"
)

var _ repository.UserRepository = (*UserRepository)(nil)

// UserRepository хранит профили в памяти и использует членство для поиска в чате.
type UserRepository struct {
	mutex   sync.RWMutex
	users   map[domain.UserID]domain.User
	members repository.ChatMemberRepository
}

func NewUserRepository(members repository.ChatMemberRepository) *UserRepository {
	return &UserRepository{
		users:   make(map[domain.UserID]domain.User),
		members: members,
	}
}

func (repo *UserRepository) Upsert(ctx context.Context, user domain.User) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	repo.mutex.Lock()
	defer repo.mutex.Unlock()

	if existing, found := repo.users[user.ID]; found {
		if existing.UpdatedAt.After(user.UpdatedAt) {
			return nil
		}
		user.CreatedAt = existing.CreatedAt
		user.Timezone = existing.Timezone
	}
	username := normalizeUsername(user.Username)
	if username != "" {
		for existingID, existing := range repo.users {
			if existingID != user.ID && normalizeUsername(existing.Username) == username {
				existing.Username = ""
				repo.users[existingID] = existing
			}
		}
	}
	repo.users[user.ID] = user
	return nil
}

func (repo *UserRepository) FindByUsernameInChat(
	ctx context.Context,
	chatID domain.ChatID,
	username string,
) (domain.User, error) {
	if err := ctx.Err(); err != nil {
		return domain.User{}, err
	}

	username = normalizeUsername(username)
	if username == "" {
		return domain.User{}, repository.ErrUserNotFound
	}
	repo.mutex.RLock()
	defer repo.mutex.RUnlock()

	for _, user := range repo.users {
		if normalizeUsername(user.Username) != username {
			continue
		}
		member, err := repo.members.Get(ctx, chatID, user.ID)
		if errors.Is(err, repository.ErrChatMemberNotFound) || err == nil && !member.IsActive() {
			return domain.User{}, repository.ErrUserNotFound
		}
		if err != nil {
			return domain.User{}, err
		}
		return user, nil
	}

	return domain.User{}, repository.ErrUserNotFound
}

func normalizeUsername(username string) string {
	return strings.ToLower(strings.TrimPrefix(strings.TrimSpace(username), "@"))
}
