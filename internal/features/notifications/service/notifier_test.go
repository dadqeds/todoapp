package notifications_service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	core_logger "github.com/dadqeds/todoapp/internal/core/logger"
	core_telegram "github.com/dadqeds/todoapp/internal/core/telegram"
)

type sent struct {
	chat   int64
	text   string
	button *core_telegram.Button
}

type fakeSender struct {
	msgs []sent
	err  error
}

func (f *fakeSender) SendMessage(_ context.Context, chat int64, text string, b *core_telegram.Button) error {
	if f.err != nil {
		return f.err
	}
	f.msgs = append(f.msgs, sent{chat, text, b})
	return nil
}

type fakeRepo struct {
	reminders  []PendingReminder
	claimed    map[int]bool
	released   []int
	recipients []DigestRecipient
	digestDays map[int]string
	tasks      []DigestTask
}

func (f *fakeRepo) GetPendingReminders(_ context.Context, horizon time.Time) ([]PendingReminder, error) {
	var out []PendingReminder
	for _, r := range f.reminders {
		if !f.claimed[r.TaskID] && !r.DueAt.After(horizon) {
			out = append(out, r)
		}
	}
	return out, nil
}

func (f *fakeRepo) ClaimReminder(_ context.Context, id int) (bool, error) {
	if f.claimed[id] {
		return false, nil
	}
	f.claimed[id] = true
	return true, nil
}

func (f *fakeRepo) ReleaseReminder(_ context.Context, id int) error {
	delete(f.claimed, id)
	f.released = append(f.released, id)
	return nil
}

func (f *fakeRepo) GetDigestRecipients(context.Context) ([]DigestRecipient, error) {
	return f.recipients, nil
}

func (f *fakeRepo) ClaimDigest(_ context.Context, userID int, day time.Time) (bool, error) {
	d := day.Format("2006-01-02")
	if f.digestDays[userID] >= d {
		return false, nil
	}
	f.digestDays[userID] = d
	return true, nil
}

func (f *fakeRepo) GetDigestTasks(context.Context, int, time.Time) ([]DigestTask, error) {
	return f.tasks, nil
}

var msk, _ = time.LoadLocation("Europe/Moscow")

func at(d, h, m int) time.Time { return time.Date(2026, 10, d, h, m, 0, 0, msk) }

func newNotifier(repo *fakeRepo, sender *fakeSender, now *time.Time) *Notifier {
	if repo.claimed == nil {
		repo.claimed = map[int]bool{}
	}
	if repo.digestDays == nil {
		repo.digestDays = map[int]string{}
	}
	n := NewNotifier(repo, sender, "todo_bot", core_logger.FromContext(context.Background()))
	n.now = func() time.Time { return *now }
	return n
}

func TestTimedReminder(t *testing.T) {
	repo := &fakeRepo{reminders: []PendingReminder{{
		TaskID: 7, Title: "Забрать <посылку>", ListTitle: "Дом", DueAt: at(5, 20, 0),
		RemindBeforeMinutes: 15, ChatID: 42, Timezone: "Europe/Moscow", DigestMinute: 540,
	}}}
	sender := &fakeSender{}
	now := at(5, 19, 40)
	n := newNotifier(repo, sender, &now)

	n.Tick(context.Background())
	if len(sender.msgs) != 0 {
		t.Fatalf("sent too early: %+v", sender.msgs)
	}

	now = at(5, 19, 45)
	n.Tick(context.Background())
	n.Tick(context.Background())
	if len(sender.msgs) != 1 {
		t.Fatalf("sent %d messages, want exactly 1", len(sender.msgs))
	}

	m := sender.msgs[0]
	for _, want := range []string{"Через 15 минут", "Забрать &lt;посылку&gt;", "Дом, сегодня в 20:00"} {
		if !strings.Contains(m.text, want) {
			t.Errorf("text %q does not contain %q", m.text, want)
		}
	}
	if m.chat != 42 || m.button == nil || m.button.URL != "https://t.me/todo_bot?startapp=task_7" {
		t.Fatalf("message = %+v", m)
	}
}

func TestMissedReminderIsDropped(t *testing.T) {
	repo := &fakeRepo{reminders: []PendingReminder{{TaskID: 1, DueAt: at(5, 10, 0), RemindBeforeMinutes: 60, ChatID: 1, Timezone: "Europe/Moscow"}}}
	sender := &fakeSender{}
	now := at(5, 12, 0)

	newNotifier(repo, sender, &now).Tick(context.Background())

	if len(sender.msgs) != 0 || !repo.claimed[1] {
		t.Fatalf("missed reminder must be claimed without sending: sent %d, claimed %v", len(sender.msgs), repo.claimed)
	}
}

func TestAllDayReminderUsesDigestTime(t *testing.T) {
	endOfDay := time.Date(2026, 10, 6, 23, 59, 59, 0, msk)
	repo := &fakeRepo{reminders: []PendingReminder{{TaskID: 2, Title: "Оплатить", ListTitle: "Личное", DueAt: endOfDay, DueAllDay: true,
		RemindBeforeMinutes: 0, ChatID: 1, Timezone: "Europe/Moscow", DigestMinute: 8 * 60}}}
	sender := &fakeSender{}
	now := at(6, 7, 59)
	n := newNotifier(repo, sender, &now)

	n.Tick(context.Background())
	if len(sender.msgs) != 0 {
		t.Fatal("sent before digest time")
	}

	now = at(6, 8, 0)
	n.Tick(context.Background())
	if len(sender.msgs) != 1 || !strings.Contains(sender.msgs[0].text, "Сегодня срок") {
		t.Fatalf("msgs = %+v", sender.msgs)
	}
}

func TestReminderDeliveryErrors(t *testing.T) {
	reminder := PendingReminder{TaskID: 3, DueAt: at(5, 20, 0), RemindBeforeMinutes: 0, ChatID: 1, Timezone: "Europe/Moscow"}
	now := at(5, 20, 0)

	repo := &fakeRepo{reminders: []PendingReminder{reminder}}
	newNotifier(repo, &fakeSender{err: errors.New("timeout")}, &now).Tick(context.Background())
	if repo.claimed[3] || len(repo.released) != 1 {
		t.Fatalf("transient error must release reminder for retry: claimed %v, released %v", repo.claimed, repo.released)
	}

	repo = &fakeRepo{reminders: []PendingReminder{reminder}}
	newNotifier(repo, &fakeSender{err: core_telegram.ErrChatUnavailable}, &now).Tick(context.Background())
	if !repo.claimed[3] || len(repo.released) != 0 {
		t.Fatalf("blocked chat must not be retried: claimed %v, released %v", repo.claimed, repo.released)
	}
}

func TestDigest(t *testing.T) {
	repo := &fakeRepo{
		recipients: []DigestRecipient{{UserID: 5, ChatID: 55, Timezone: "Europe/Moscow", DigestMinute: 7*60 + 30}},
		tasks: []DigestTask{
			{Title: "Оплатить интернет", DueAt: time.Date(2026, 10, 4, 23, 59, 59, 0, msk), DueAllDay: true},
			{Title: "Забрать посылку", DueAt: at(5, 18, 0)},
			{Title: "Позвонить маме", DueAt: time.Date(2026, 10, 5, 23, 59, 59, 0, msk), DueAllDay: true},
		},
	}
	sender := &fakeSender{}
	now := at(5, 7, 29)
	n := newNotifier(repo, sender, &now)

	n.Tick(context.Background())
	if len(sender.msgs) != 0 {
		t.Fatal("digest sent before its time")
	}

	now = at(5, 7, 30)
	n.Tick(context.Background())
	now = at(5, 7, 31)
	n.Tick(context.Background())
	if len(sender.msgs) != 1 {
		t.Fatalf("sent %d digests, want 1 per day", len(sender.msgs))
	}

	text := sender.msgs[0].text
	for _, want := range []string{"На сегодня 2 задачи", "Забрать посылку — 18:00", "Позвонить маме", "Просрочено", "Оплатить интернет (вчера)"} {
		if !strings.Contains(text, want) {
			t.Errorf("digest %q does not contain %q", text, want)
		}
	}
	if sender.msgs[0].button.URL != "https://t.me/todo_bot?startapp" {
		t.Fatalf("button = %+v", sender.msgs[0].button)
	}

	now = at(6, 7, 30)
	n.Tick(context.Background())
	if len(sender.msgs) != 2 {
		t.Fatal("next day digest not sent")
	}
}

func TestDigestSkippedWhenEmptyOrTooLate(t *testing.T) {
	repo := &fakeRepo{recipients: []DigestRecipient{{UserID: 1, ChatID: 1, Timezone: "Europe/Moscow", DigestMinute: 9 * 60}}}
	sender := &fakeSender{}
	now := at(5, 9, 0)
	newNotifier(repo, sender, &now).Tick(context.Background())
	if len(sender.msgs) != 0 {
		t.Fatal("empty digest must not be sent")
	}

	repo = &fakeRepo{
		recipients: []DigestRecipient{{UserID: 1, ChatID: 1, Timezone: "Europe/Moscow", DigestMinute: 9 * 60}},
		tasks:      []DigestTask{{Title: "x", DueAt: at(5, 20, 0)}},
	}
	now = at(5, 12, 1)
	newNotifier(repo, sender, &now).Tick(context.Background())
	if len(sender.msgs) != 0 {
		t.Fatal("digest more than 3 hours late must be skipped")
	}
}

func TestDueText(t *testing.T) {
	now := at(5, 12, 0)
	tests := []struct {
		due    time.Time
		allDay bool
		want   string
	}{
		{at(5, 20, 0), false, "сегодня в 20:00"},
		{at(6, 10, 0), false, "завтра в 10:00"},
		{time.Date(2026, 10, 4, 23, 59, 59, 0, msk), true, "вчера"},
		{at(18, 9, 30), false, "18 октября в 09:30"},
	}
	for _, tt := range tests {
		if got := dueText(tt.due, tt.allDay, now, msk); got != tt.want {
			t.Errorf("dueText(%v) = %q, want %q", tt.due, got, tt.want)
		}
	}
}

func TestGreeting(t *testing.T) {
	for hour, want := range map[int]string{3: "Доброй ночи", 7: "Доброе утро", 13: "Добрый день", 20: "Добрый вечер"} {
		if got := greeting(hour); got != want {
			t.Errorf("greeting(%d) = %q, want %q", hour, got, want)
		}
	}
}
