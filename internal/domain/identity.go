package domain

// UserID — постоянный числовой идентификатор пользователя Telegram.
type UserID int64

// IsValid сообщает, может ли значение быть ID пользователя Telegram.
func (id UserID) IsValid() bool {
	return id > 0
}

// ChatID — постоянный числовой идентификатор чата Telegram.
// У групповых чатов это значение может быть отрицательным.
type ChatID int64

// IsValid сообщает, задан ли идентификатор чата.
func (id ChatID) IsValid() bool {
	return id != 0
}
