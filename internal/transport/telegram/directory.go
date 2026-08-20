package telegram

import (
	"strings"
	"sync"

	"github.com/elt00n/taskflask-telegram-bot/internal/domain"
)

type directoryKey struct {
	chatID   domain.ChatID
	username string
}

// MemberDirectory временно сопоставляет usernames с ID известных участников.
// После подключения PostgreSQL эти данные будут храниться постоянно.
type MemberDirectory struct {
	mutex sync.RWMutex
	users map[directoryKey]domain.UserID
}

func NewMemberDirectory() *MemberDirectory {
	return &MemberDirectory{users: make(map[directoryKey]domain.UserID)}
}

func (directory *MemberDirectory) Observe(
	chatID domain.ChatID,
	userID domain.UserID,
	username string,
) {
	username = normalizeUsername(username)
	if username == "" {
		return
	}

	directory.mutex.Lock()
	defer directory.mutex.Unlock()
	directory.users[directoryKey{chatID: chatID, username: username}] = userID
}

func (directory *MemberDirectory) Resolve(
	chatID domain.ChatID,
	usernames []string,
) ([]domain.UserID, []string) {
	directory.mutex.RLock()
	defer directory.mutex.RUnlock()

	userIDs := make([]domain.UserID, 0, len(usernames))
	unknown := make([]string, 0)
	for _, username := range usernames {
		normalized := normalizeUsername(username)
		userID, exists := directory.users[directoryKey{chatID: chatID, username: normalized}]
		if !exists {
			unknown = append(unknown, normalized)
			continue
		}
		userIDs = append(userIDs, userID)
	}

	return userIDs, unknown
}

func normalizeUsername(username string) string {
	return strings.ToLower(strings.TrimPrefix(strings.TrimSpace(username), "@"))
}
