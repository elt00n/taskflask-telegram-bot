// Package parser преобразует пользовательский текст в команды приложения.
package parser

import (
	"errors"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/elt00n/taskflask-telegram-bot/internal/domain"
)

var (
	ErrTaskCommandRequired = errors.New("task command is required")
	ErrTaskTitleRequired   = errors.New("task title is required")
	ErrReferenceTimeNeeded = errors.New("reference time is required")
	ErrInvalidTaskTime     = errors.New("invalid task time")
	ErrInvalidTaskDate     = errors.New("invalid task date")
	ErrTaskTimeInPast      = errors.New("task time is in the past")
)

var (
	taskCommandPattern  = regexp.MustCompile(`^/task(?:@[A-Za-z0-9_]+)?(?:\s+|$)`)
	mentionPattern      = regexp.MustCompile(`@[A-Za-z0-9_]{3,32}`)
	relativeTimePattern = regexp.MustCompile(
		`(?i)^(сегодня|завтра)(?:\s+в)?\s+(\d{1,2}:\d{2})\s+(.+)$`,
	)
	weekdayTimePattern = regexp.MustCompile(
		`(?i)^в\s+([а-яё]+)(?:\s+в)?\s+(\d{1,2}:\d{2})\s+(.+)$`,
	)
	deadlineDatePattern = regexp.MustCompile(
		`(?i)^до\s+(\d{1,2})\s+([а-яё]+)(?:\s+(\d{4}))?(?:\s+(\d{1,2}:\d{2}))?\s+(.+)$`,
	)
	deadlineWeekdayPattern = regexp.MustCompile(
		`(?i)^до\s+([а-яё]+)(?:\s+(\d{1,2}:\d{2}))?\s+(.+)$`,
	)
	deadlinePrefixPattern = regexp.MustCompile(`(?i)^до(?:\s+|$)`)
)

var russianMonths = map[string]time.Month{
	"января":   time.January,
	"февраля":  time.February,
	"марта":    time.March,
	"апреля":   time.April,
	"мая":      time.May,
	"июня":     time.June,
	"июля":     time.July,
	"августа":  time.August,
	"сентября": time.September,
	"октября":  time.October,
	"ноября":   time.November,
	"декабря":  time.December,
}

var russianWeekdays = map[string]time.Weekday{
	"понедельник":  time.Monday,
	"понедельника": time.Monday,
	"вторник":      time.Tuesday,
	"вторника":     time.Tuesday,
	"среду":        time.Wednesday,
	"среды":        time.Wednesday,
	"четверг":      time.Thursday,
	"четверга":     time.Thursday,
	"пятницу":      time.Friday,
	"пятницы":      time.Friday,
	"субботу":      time.Saturday,
	"субботы":      time.Saturday,
	"воскресенье":  time.Sunday,
	"воскресенья":  time.Sunday,
}

// ParsedTask содержит данные, извлечённые из команды /task.
// Usernames позднее будут сопоставлены с Telegram ID известных участников чата.
type ParsedTask struct {
	Title                string
	StartAt              *time.Time
	Deadline             *time.Time
	Priority             domain.TaskPriority
	ParticipantUsernames []string
}

// ParseTask разбирает детерминированные русскоязычные шаблоны команды /task.
func ParseTask(text string, now time.Time, location *time.Location) (ParsedTask, error) {
	text = strings.TrimSpace(text)
	command := taskCommandPattern.FindString(text)
	if command == "" {
		return ParsedTask{}, ErrTaskCommandRequired
	}
	if now.IsZero() {
		return ParsedTask{}, ErrReferenceTimeNeeded
	}
	if location == nil {
		location = time.UTC
	}

	body := strings.TrimSpace(strings.TrimPrefix(text, command))
	body, usernames := extractUsernames(body)
	body, priority := extractPriority(body)
	body = normalizeSpaces(body)

	parsed := ParsedTask{
		Priority:             priority,
		ParticipantUsernames: usernames,
	}
	now = now.In(location)

	var err error
	switch {
	case relativeTimePattern.MatchString(body):
		parsed.Title, parsed.StartAt, err = parseRelativeTime(body, now, location)
	case weekdayTimePattern.MatchString(body):
		parsed.Title, parsed.StartAt, err = parseWeekdayTime(body, now, location)
	case deadlineDatePattern.MatchString(body):
		parsed.Title, parsed.Deadline, err = parseDeadlineDate(body, now, location)
	case deadlineWeekdayPattern.MatchString(body):
		parsed.Title, parsed.Deadline, err = parseDeadlineWeekday(body, now, location)
	case deadlinePrefixPattern.MatchString(body):
		err = ErrInvalidTaskDate
	default:
		parsed.Title = body
	}
	if err != nil {
		return ParsedTask{}, err
	}
	if strings.TrimSpace(parsed.Title) == "" {
		return ParsedTask{}, ErrTaskTitleRequired
	}

	parsed.Title = strings.TrimSpace(parsed.Title)
	return parsed, nil
}

func extractUsernames(text string) (string, []string) {
	matches := mentionPattern.FindAllString(text, -1)
	usernames := make([]string, 0, len(matches))
	seen := make(map[string]struct{}, len(matches))

	for _, match := range matches {
		username := strings.ToLower(strings.TrimPrefix(match, "@"))
		if _, duplicate := seen[username]; duplicate {
			continue
		}
		seen[username] = struct{}{}
		usernames = append(usernames, username)
	}

	return mentionPattern.ReplaceAllString(text, " "), usernames
}

func extractPriority(text string) (string, domain.TaskPriority) {
	priority := domain.TaskPriorityNormal
	keptWords := make([]string, 0)

	for _, word := range strings.Fields(text) {
		normalized := strings.ToLower(strings.Trim(word, ".,!?:;"))
		switch normalized {
		case "важно":
			if priority < domain.TaskPriorityHigh {
				priority = domain.TaskPriorityHigh
			}
		case "срочно":
			priority = domain.TaskPriorityCritical
		default:
			keptWords = append(keptWords, word)
		}
	}

	return strings.Join(keptWords, " "), priority
}

func parseRelativeTime(
	body string,
	now time.Time,
	location *time.Location,
) (string, *time.Time, error) {
	parts := relativeTimePattern.FindStringSubmatch(body)
	hour, minute, err := parseClock(parts[2])
	if err != nil {
		return "", nil, err
	}

	dayOffset := 0
	if strings.EqualFold(parts[1], "завтра") {
		dayOffset = 1
	}
	date := now.AddDate(0, 0, dayOffset)
	startAt := time.Date(date.Year(), date.Month(), date.Day(), hour, minute, 0, 0, location)
	if !startAt.After(now) {
		return "", nil, ErrTaskTimeInPast
	}

	return parts[3], &startAt, nil
}

func parseWeekdayTime(
	body string,
	now time.Time,
	location *time.Location,
) (string, *time.Time, error) {
	parts := weekdayTimePattern.FindStringSubmatch(body)
	weekday, supported := russianWeekdays[strings.ToLower(parts[1])]
	if !supported {
		return "", nil, ErrInvalidTaskDate
	}
	hour, minute, err := parseClock(parts[2])
	if err != nil {
		return "", nil, err
	}

	startAt := nextWeekdayTime(now, weekday, hour, minute, location)
	return parts[3], &startAt, nil
}

func parseDeadlineDate(
	body string,
	now time.Time,
	location *time.Location,
) (string, *time.Time, error) {
	parts := deadlineDatePattern.FindStringSubmatch(body)
	day, err := strconv.Atoi(parts[1])
	if err != nil {
		return "", nil, ErrInvalidTaskDate
	}
	month, supported := russianMonths[strings.ToLower(parts[2])]
	if !supported {
		return "", nil, ErrInvalidTaskDate
	}

	year := now.Year()
	explicitYear := parts[3] != ""
	if explicitYear {
		year, err = strconv.Atoi(parts[3])
		if err != nil {
			return "", nil, ErrInvalidTaskDate
		}
	}
	hour, minute := 23, 59
	if parts[4] != "" {
		hour, minute, err = parseClock(parts[4])
		if err != nil {
			return "", nil, err
		}
	}

	deadline, valid := calendarDate(year, month, day, hour, minute, location)
	if !valid {
		return "", nil, ErrInvalidTaskDate
	}
	if deadline.Before(now) && !explicitYear {
		deadline, valid = calendarDate(year+1, month, day, hour, minute, location)
		if !valid {
			return "", nil, ErrInvalidTaskDate
		}
	}
	if deadline.Before(now) {
		return "", nil, ErrTaskTimeInPast
	}

	return parts[5], &deadline, nil
}

func parseDeadlineWeekday(
	body string,
	now time.Time,
	location *time.Location,
) (string, *time.Time, error) {
	parts := deadlineWeekdayPattern.FindStringSubmatch(body)
	weekday, supported := russianWeekdays[strings.ToLower(parts[1])]
	if !supported {
		return "", nil, ErrInvalidTaskDate
	}

	hour, minute := 23, 59
	var err error
	if parts[2] != "" {
		hour, minute, err = parseClock(parts[2])
		if err != nil {
			return "", nil, err
		}
	}

	deadline := nextWeekdayTime(now, weekday, hour, minute, location)
	return parts[3], &deadline, nil
}

func parseClock(value string) (int, int, error) {
	parsed, err := time.Parse("15:04", value)
	if err != nil {
		return 0, 0, ErrInvalidTaskTime
	}
	return parsed.Hour(), parsed.Minute(), nil
}

func nextWeekdayTime(
	now time.Time,
	weekday time.Weekday,
	hour int,
	minute int,
	location *time.Location,
) time.Time {
	daysAhead := (int(weekday) - int(now.Weekday()) + 7) % 7
	date := now.AddDate(0, 0, daysAhead)
	result := time.Date(date.Year(), date.Month(), date.Day(), hour, minute, 0, 0, location)
	if !result.After(now) {
		result = result.AddDate(0, 0, 7)
	}
	return result
}

func calendarDate(
	year int,
	month time.Month,
	day int,
	hour int,
	minute int,
	location *time.Location,
) (time.Time, bool) {
	result := time.Date(year, month, day, hour, minute, 0, 0, location)
	valid := result.Year() == year && result.Month() == month && result.Day() == day
	return result, valid
}

func normalizeSpaces(value string) string {
	return strings.Join(strings.Fields(value), " ")
}
