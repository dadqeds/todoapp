package domain

import (
	"errors"
	"testing"
	"time"

	core_errors "github.com/dadqeds/todoapp/internal/core/errors"
)

func mustLoc(t *testing.T, name string) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation(name)
	if err != nil {
		t.Fatal(err)
	}
	return loc
}

func TestRecurrenceNext(t *testing.T) {
	msk := mustLoc(t, "Europe/Moscow")
	at := func(y int, m time.Month, d, h, min int) time.Time { return time.Date(y, m, d, h, min, 0, 0, msk) }
	// «Сейчас» раньше всех сроков, чтобы проверять один шаг.
	past := at(2020, 1, 1, 0, 0)

	tests := []struct {
		name string
		rule Recurrence
		due  time.Time
		now  time.Time
		want time.Time
	}{
		{"daily keeps time", Recurrence{Kind: RepeatDaily}, at(2026, 10, 5, 20, 0), past, at(2026, 10, 6, 20, 0)},
		{"weekly mon,thu from mon", Recurrence{Kind: RepeatWeekly, Weekdays: []int{1, 4}}, at(2026, 10, 5, 20, 0), past, at(2026, 10, 8, 20, 0)},
		{"weekly mon,thu from thu", Recurrence{Kind: RepeatWeekly, Weekdays: []int{1, 4}}, at(2026, 10, 8, 20, 0), past, at(2026, 10, 12, 20, 0)},
		{"weekly sunday", Recurrence{Kind: RepeatWeekly, Weekdays: []int{7}}, at(2026, 10, 4, 9, 0), past, at(2026, 10, 11, 9, 0)},
		{"monthly 31 to feb", Recurrence{Kind: RepeatMonthly, MonthDay: 31}, at(2026, 1, 31, 10, 0), past, at(2026, 2, 28, 10, 0)},
		{"monthly back to 31", Recurrence{Kind: RepeatMonthly, MonthDay: 31}, at(2026, 2, 28, 10, 0), past, at(2026, 3, 31, 10, 0)},
		{"yearly feb 29 non-leap", Recurrence{Kind: RepeatYearly, Month: 2, MonthDay: 29}, at(2028, 2, 29, 10, 0), past, at(2029, 2, 28, 10, 0)},
		{"yearly back to feb 29", Recurrence{Kind: RepeatYearly, Month: 2, MonthDay: 29}, at(2031, 2, 28, 10, 0), past, at(2032, 2, 29, 10, 0)},
		{"late completion skips past dates", Recurrence{Kind: RepeatDaily}, at(2026, 10, 1, 9, 0), at(2026, 10, 5, 12, 0), at(2026, 10, 6, 9, 0)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.rule.Next(tt.due, tt.now, msk)
			if !got.Equal(tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}
}

// Срок в понедельник 23:30 по Москве — это понедельник 20:30 UTC, но для
// пользователя из Владивостока это уже вторник. День недели считается по
// поясу пользователя.
func TestRecurrenceUsesUserTimezone(t *testing.T) {
	msk := mustLoc(t, "Europe/Moscow")
	vvo := mustLoc(t, "Asia/Vladivostok")
	due := time.Date(2026, 10, 5, 23, 30, 0, 0, msk) // пн по Москве
	past := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	rule := Recurrence{Kind: RepeatWeekly, Weekdays: []int{1}}

	if got := rule.Next(due, past, msk); got.In(msk).Weekday() != time.Monday {
		t.Fatalf("msk: next = %v, want monday", got.In(msk))
	}
	if got := rule.Next(due, past, vvo); got.In(vvo).Weekday() != time.Monday {
		t.Fatalf("vvo: next = %v, want monday in Vladivostok", got.In(vvo))
	}
}

func TestRecurrenceParseFormat(t *testing.T) {
	for _, s := range []string{"daily", "weekly:1,4", "monthly:31", "yearly:02-29"} {
		r, err := ParseRecurrence(s)
		if err != nil {
			t.Fatalf("%s: %v", s, err)
		}
		if r.String() != s {
			t.Fatalf("roundtrip %s -> %s", s, r.String())
		}
	}

	for _, s := range []string{"", "hourly", "weekly:", "weekly:0", "weekly:8", "monthly:x"} {
		if _, err := ParseRecurrence(s); !errors.Is(err, core_errors.ErrInvalidArgument) {
			t.Errorf("%q: err = %v, want ErrInvalidArgument", s, err)
		}
	}
}

func TestRecurrenceAnchored(t *testing.T) {
	msk := mustLoc(t, "Europe/Moscow")
	due := time.Date(2026, 1, 31, 23, 0, 0, 0, msk) // 31-е по Москве, 20:00 UTC

	r := Recurrence{Kind: RepeatMonthly}.Anchored(due, msk)
	if r.MonthDay != 31 {
		t.Fatalf("anchor = %d, want 31", r.MonthDay)
	}

	w := Recurrence{Kind: RepeatWeekly, Weekdays: []int{4, 1, 4}}.Anchored(due, msk)
	if len(w.Weekdays) != 2 || w.Weekdays[0] != 1 || w.Weekdays[1] != 4 {
		t.Fatalf("weekdays = %v, want [1 4]", w.Weekdays)
	}
}

func TestTaskRepeatRequiresDue(t *testing.T) {
	task := Task{Title: "a", CreatedAt: time.Now(), Repeat: &Recurrence{Kind: RepeatDaily}}
	if err := task.Validate(); !errors.Is(err, core_errors.ErrInvalidArgument) {
		t.Fatalf("err = %v, want ErrInvalidArgument", err)
	}

	due := time.Now().Add(time.Hour)
	task.DueAt = &due
	if err := task.ApplyPatch(TaskPatch{DueAt: setNull[time.Time]()}); err != nil {
		t.Fatal(err)
	}
	if task.Repeat != nil {
		t.Fatal("clearing due must clear repeat")
	}
}

func TestTaskNextOccurrence(t *testing.T) {
	msk := mustLoc(t, "Europe/Moscow")
	due := time.Date(2026, 10, 5, 20, 0, 0, 0, msk)
	desc := "пакеты у двери"
	task := Task{ID: 7, Title: "Вынести мусор", Description: &desc, Completed: true, AuthorUserID: 3, ListID: 9, DueAt: &due,
		Repeat: &Recurrence{Kind: RepeatWeekly, Weekdays: []int{1, 4}}}

	next, ok := task.NextOccurrence(time.Date(2026, 10, 5, 21, 0, 0, 0, msk), msk)
	if !ok {
		t.Fatal("no next occurrence")
	}
	if next.ID != UninitializedID || next.Completed || next.Title != task.Title || next.ListID != 9 || next.AuthorUserID != 3 {
		t.Fatalf("next = %+v", next)
	}
	if want := time.Date(2026, 10, 8, 20, 0, 0, 0, msk); !next.DueAt.Equal(want) {
		t.Fatalf("next due = %v, want %v", next.DueAt, want)
	}
	if next.Repeat == nil || next.Repeat == task.Repeat {
		t.Fatal("next must carry its own copy of the repeat rule")
	}

	if _, ok := (&Task{Title: "a"}).NextOccurrence(time.Now(), msk); ok {
		t.Fatal("non-repeating task must not have next occurrence")
	}
}

func TestUserTimezone(t *testing.T) {
	u := User{FullName: "Ян", Timezone: "Europe/Moscow"}
	if err := u.Validate(); err != nil {
		t.Fatal(err)
	}
	if u.Location().String() != "Europe/Moscow" {
		t.Fatalf("location = %v", u.Location())
	}

	u.Timezone = "Mars/Olympus"
	if err := u.Validate(); !errors.Is(err, core_errors.ErrInvalidArgument) {
		t.Fatalf("bad tz: err = %v", err)
	}
	if (&User{}).Location() != time.UTC {
		t.Fatal("empty tz must be UTC")
	}
}
