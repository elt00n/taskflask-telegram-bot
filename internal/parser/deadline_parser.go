package parser

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	deadlineValueDatePattern = regexp.MustCompile(
		`(?i)^(\d{1,2})\s+([а-яё]+)(?:\s+(\d{4}))?(?:\s+(\d{1,2}:\d{2}))?$`,
	)
	deadlineValueWeekdayPattern = regexp.MustCompile(
		`(?i)^([а-яё]+)(?:\s+(\d{1,2}:\d{2}))?$`,
	)
)

// ParseDeadline разбирает значение для команды изменения дедлайна.
// Поддерживаются календарная дата и ближайший день недели.
func ParseDeadline(
	value string,
	now time.Time,
	location *time.Location,
) (*time.Time, error) {
	if now.IsZero() {
		return nil, ErrReferenceTimeNeeded
	}
	if location == nil {
		location = time.UTC
	}

	value = strings.TrimSpace(value)
	value = strings.TrimSpace(strings.TrimPrefix(strings.ToLower(value), "до "))
	now = now.In(location)

	if parts := deadlineValueDatePattern.FindStringSubmatch(value); parts != nil {
		return parseDeadlineDateValue(parts, now, location)
	}
	if parts := deadlineValueWeekdayPattern.FindStringSubmatch(value); parts != nil {
		weekday, supported := russianWeekdays[parts[1]]
		if !supported {
			return nil, ErrInvalidTaskDate
		}

		hour, minute := 23, 59
		var err error
		if parts[2] != "" {
			hour, minute, err = parseClock(parts[2])
			if err != nil {
				return nil, err
			}
		}
		deadline := nextWeekdayTime(now, weekday, hour, minute, location)
		return &deadline, nil
	}

	return nil, ErrInvalidTaskDate
}

func parseDeadlineDateValue(
	parts []string,
	now time.Time,
	location *time.Location,
) (*time.Time, error) {
	day, err := strconv.Atoi(parts[1])
	if err != nil {
		return nil, ErrInvalidTaskDate
	}
	month, supported := russianMonths[parts[2]]
	if !supported {
		return nil, ErrInvalidTaskDate
	}

	year := now.Year()
	explicitYear := parts[3] != ""
	if explicitYear {
		year, err = strconv.Atoi(parts[3])
		if err != nil {
			return nil, ErrInvalidTaskDate
		}
	}
	hour, minute := 23, 59
	if parts[4] != "" {
		hour, minute, err = parseClock(parts[4])
		if err != nil {
			return nil, err
		}
	}

	deadline, valid := calendarDate(year, month, day, hour, minute, location)
	if !valid {
		return nil, ErrInvalidTaskDate
	}
	if deadline.Before(now) && !explicitYear {
		deadline, valid = calendarDate(year+1, month, day, hour, minute, location)
		if !valid {
			return nil, ErrInvalidTaskDate
		}
	}
	if deadline.Before(now) {
		return nil, ErrTaskTimeInPast
	}
	return &deadline, nil
}
