package domain

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	core_errors "github.com/dadqeds/todoapp/internal/core/errors"
)

type RepeatKind string

const (
	RepeatDaily   RepeatKind = "daily"
	RepeatWeekly  RepeatKind = "weekly"
	RepeatMonthly RepeatKind = "monthly"
	RepeatYearly  RepeatKind = "yearly"
)

// Recurrence — правило повтора задачи. Дни недели в формате ISO: 1 = пн … 7 = вс.
// MonthDay и Month — «якорь» месячного и годового повтора: день из исходного
// срока. Он нужен, чтобы задача на 31-е после февраля вернулась на 31-е, а не
// осталась на 28-м.
type Recurrence struct {
	Kind     RepeatKind
	Weekdays []int
	MonthDay int
	Month    time.Month
}

func (r Recurrence) Validate() error {
	switch r.Kind {
	case RepeatDaily, RepeatYearly, RepeatMonthly:
		return nil
	case RepeatWeekly:
		if len(r.Weekdays) == 0 {
			return fmt.Errorf("weekly repeat needs at least one weekday: %w", core_errors.ErrInvalidArgument)
		}
		for _, d := range r.Weekdays {
			if d < 1 || d > 7 {
				return fmt.Errorf("invalid weekday %d: %w", d, core_errors.ErrInvalidArgument)
			}
		}
		return nil
	default:
		return fmt.Errorf("unknown repeat kind %q: %w", r.Kind, core_errors.ErrInvalidArgument)
	}
}

// Anchored возвращает правило с якорем, взятым из срока по местному времени.
func (r Recurrence) Anchored(due time.Time, loc *time.Location) Recurrence {
	local := due.In(loc)
	out := r
	out.Weekdays = normalizeWeekdays(r.Weekdays)
	switch r.Kind {
	case RepeatMonthly:
		out.MonthDay = local.Day()
	case RepeatYearly:
		out.MonthDay = local.Day()
		out.Month = local.Month()
	}
	return out
}

func normalizeWeekdays(days []int) []int {
	out := slices.Clone(days)
	slices.Sort(out)
	return slices.Compact(out)
}

// String — формат хранения в БД: daily | weekly:1,4 | monthly:31 | yearly:02-29.
func (r Recurrence) String() string {
	switch r.Kind {
	case RepeatWeekly:
		parts := make([]string, len(r.Weekdays))
		for i, d := range r.Weekdays {
			parts[i] = strconv.Itoa(d)
		}
		return "weekly:" + strings.Join(parts, ",")
	case RepeatMonthly:
		return fmt.Sprintf("monthly:%d", r.MonthDay)
	case RepeatYearly:
		return fmt.Sprintf("yearly:%02d-%02d", int(r.Month), r.MonthDay)
	default:
		return string(r.Kind)
	}
}

func ParseRecurrence(s string) (Recurrence, error) {
	kind, arg, _ := strings.Cut(s, ":")
	r := Recurrence{Kind: RepeatKind(kind)}

	var err error
	switch r.Kind {
	case RepeatWeekly:
		for _, p := range strings.Split(arg, ",") {
			var d int
			if d, err = strconv.Atoi(p); err != nil {
				break
			}
			r.Weekdays = append(r.Weekdays, d)
		}
	case RepeatMonthly:
		r.MonthDay, err = strconv.Atoi(arg)
	case RepeatYearly:
		var m, d int
		_, err = fmt.Sscanf(arg, "%d-%d", &m, &d)
		r.Month, r.MonthDay = time.Month(m), d
	}
	if err != nil {
		return Recurrence{}, fmt.Errorf("parse repeat rule %q: %v: %w", s, err, core_errors.ErrInvalidArgument)
	}

	if err := r.Validate(); err != nil {
		return Recurrence{}, err
	}
	return r, nil
}

// Next возвращает следующий срок после due, который уже позже now: если
// задачу закрыли с опозданием, следующая не окажется сразу просроченной.
// Время суток сохраняется по местному времени пользователя.
func (r Recurrence) Next(due time.Time, now time.Time, loc *time.Location) time.Time {
	next := r.step(due.In(loc))
	for !next.After(now) {
		next = r.step(next)
	}
	return next
}

func (r Recurrence) step(t time.Time) time.Time {
	switch r.Kind {
	case RepeatWeekly:
		for i := 1; i <= 7; i++ {
			c := t.AddDate(0, 0, i)
			if slices.Contains(r.Weekdays, isoWeekday(c)) {
				return c
			}
		}
		return t.AddDate(0, 0, 7)
	case RepeatMonthly:
		y, m, _ := t.Date()
		return clampedDate(y, m+1, r.MonthDay, t)
	case RepeatYearly:
		return clampedDate(t.Year()+1, r.Month, r.MonthDay, t)
	default:
		return t.AddDate(0, 0, 1)
	}
}

func isoWeekday(t time.Time) int {
	if t.Weekday() == time.Sunday {
		return 7
	}
	return int(t.Weekday())
}

// clampedDate — дата с днём day, но не позже последнего дня месяца; время
// суток и пояс берутся из clock.
func clampedDate(year int, month time.Month, day int, clock time.Time) time.Time {
	first := time.Date(year, month, 1, 0, 0, 0, 0, clock.Location())
	last := first.AddDate(0, 1, -1).Day()
	if day > last {
		day = last
	}
	return time.Date(first.Year(), first.Month(), day, clock.Hour(), clock.Minute(), clock.Second(), 0, clock.Location())
}
