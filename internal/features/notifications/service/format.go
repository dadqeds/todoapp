package notifications_service

import (
	"fmt"
	"time"
)

var monthsGenitive = [...]string{"января", "февраля", "марта", "апреля", "мая", "июня",
	"июля", "августа", "сентября", "октября", "ноября", "декабря"}

func plural(n int, one, few, many string) string {
	m10, m100 := n%10, n%100
	switch {
	case m10 == 1 && m100 != 11:
		return one
	case m10 >= 2 && m10 <= 4 && (m100 < 12 || m100 > 14):
		return few
	default:
		return many
	}
}

// dueText — срок по-русски относительно сегодняшнего дня пользователя:
// «сегодня в 20:00», «завтра», «вчера», «8 октября в 10:00».
func dueText(due time.Time, allDay bool, now time.Time, loc *time.Location) string {
	d := due.In(loc)
	n := now.In(loc)
	// Даты сравниваем в UTC, чтобы сутки с переводом часов не сбивали счёт.
	day := time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, time.UTC)
	today := time.Date(n.Year(), n.Month(), n.Day(), 0, 0, 0, 0, time.UTC)

	var text string
	switch int(day.Sub(today).Hours() / 24) {
	case 0:
		text = "сегодня"
	case 1:
		text = "завтра"
	case -1:
		text = "вчера"
	default:
		text = fmt.Sprintf("%d %s", d.Day(), monthsGenitive[d.Month()-1])
	}

	if !allDay {
		text += " в " + d.Format("15:04")
	}
	return text
}
