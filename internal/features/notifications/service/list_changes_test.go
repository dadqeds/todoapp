package notifications_service

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	core_logger "github.com/dadqeds/todoapp/internal/core/logger"
	core_telegram "github.com/dadqeds/todoapp/internal/core/telegram"
)

// Фейковая очередь: отдаёт группы, у которых есть неотправленные события
// не позже readyBefore, как это делает SQL в репозитории.
func (f *fakeRepo) GetPendingListChanges(_ context.Context, readyBefore time.Time) ([]ListChangeBatch, error) {
	f.readyBefore = readyBefore
	var out []ListChangeBatch
	for _, b := range f.batches {
		var pending []ListChange
		for _, c := range b.Changes {
			if !f.sentChanges[c.ID] {
				pending = append(pending, c)
			}
		}
		if len(pending) > 0 && !pending[0].CreatedAt.After(readyBefore) {
			b.Changes = pending
			out = append(out, b)
		}
	}
	return out, nil
}

func (f *fakeRepo) ClaimListChanges(_ context.Context, ids []int64) (bool, error) {
	if f.sentChanges == nil {
		f.sentChanges = map[int64]bool{}
	}
	claimed := false
	for _, id := range ids {
		if !f.sentChanges[id] {
			f.sentChanges[id] = true
			claimed = true
		}
	}
	return claimed, nil
}

func (f *fakeRepo) ReleaseListChanges(_ context.Context, ids []int64) error {
	for _, id := range ids {
		delete(f.sentChanges, id)
	}
	f.releasedChanges = append(f.releasedChanges, ids...)
	return nil
}

func (f *fakeRepo) DeleteListChangesBefore(_ context.Context, before time.Time) error {
	f.cleanedTo = before
	return nil
}

func homeBatch() ListChangeBatch {
	return ListChangeBatch{
		ListID: 3, ListTitle: "Дом <семья>", RecipientUserID: 7, ChatID: 77,
		Changes: []ListChange{
			{ID: 1, ActorName: "Анна", Kind: "task_added", TaskTitle: "Купить молоко", CreatedAt: at(5, 10, 0)},
			{ID: 2, ActorName: "Пётр", Kind: "task_completed", TaskTitle: "Вынести мусор", CreatedAt: at(5, 10, 1)},
			{ID: 3, ActorName: "Анна", Kind: "task_added", TaskTitle: "Позвонить <маме>", CreatedAt: at(5, 10, 2)},
		},
	}
}

func TestListChangesWaitThreeMinutesAndGoInOneMessage(t *testing.T) {
	repo := &fakeRepo{batches: []ListChangeBatch{homeBatch()}}
	sender := &fakeSender{}
	now := at(5, 10, 2)
	n := newNotifier(repo, sender, &now)

	n.Tick(context.Background())
	if len(sender.msgs) != 0 {
		t.Fatalf("sent before 3 minutes passed: %+v", sender.msgs)
	}
	if !repo.readyBefore.Equal(at(5, 9, 59)) {
		t.Fatalf("readyBefore = %v, want now - 3m", repo.readyBefore)
	}

	now = at(5, 10, 3)
	n.Tick(context.Background())
	n.Tick(context.Background())
	if len(sender.msgs) != 1 {
		t.Fatalf("sent %d messages, want exactly 1", len(sender.msgs))
	}

	m := sender.msgs[0]
	for _, want := range []string{
		"Изменения в списке «Дом &lt;семья&gt;»",
		"<b>Добавлены</b>\n• Купить молоко — Анна\n• Позвонить &lt;маме&gt; — Анна",
		"<b>Выполнены</b>\n• Вынести мусор — Пётр",
	} {
		if !strings.Contains(m.text, want) {
			t.Errorf("text %q does not contain %q", m.text, want)
		}
	}
	if m.chat != 77 || m.button == nil || m.button.Text != "Открыть список" || m.button.URL != "https://t.me/todo_bot?startapp=list_3" {
		t.Fatalf("message = %+v", m)
	}
	if !repo.cleanedTo.Equal(now.Add(-24 * time.Hour)) {
		t.Fatalf("queue cleaned to %v", repo.cleanedTo)
	}
}

func TestListChangesDeliveryErrors(t *testing.T) {
	now := at(5, 11, 0)

	repo := &fakeRepo{batches: []ListChangeBatch{homeBatch()}}
	newNotifier(repo, &fakeSender{err: errors.New("timeout")}, &now).Tick(context.Background())
	if len(repo.releasedChanges) != 3 || repo.sentChanges[1] {
		t.Fatalf("transient error must release changes for retry: released %v", repo.releasedChanges)
	}

	repo = &fakeRepo{batches: []ListChangeBatch{homeBatch()}}
	newNotifier(repo, &fakeSender{err: core_telegram.ErrChatUnavailable}, &now).Tick(context.Background())
	if len(repo.releasedChanges) != 0 || !repo.sentChanges[1] {
		t.Fatalf("blocked chat must not be retried: released %v", repo.releasedChanges)
	}
}

func TestListChangesTextIsShortened(t *testing.T) {
	b := ListChangeBatch{ListTitle: "Работа"}
	for i := range 13 {
		b.Changes = append(b.Changes, ListChange{ID: int64(i), ActorName: "Анна", Kind: "task_added", TaskTitle: "задача"})
	}

	text := listChangesText(b)
	if strings.Count(text, "• задача") != listChangesShown || !strings.Contains(text, "…и ещё 3 задачи") {
		t.Fatalf("text = %q", text)
	}
}

// Отправка через настоящий клиент Bot API в фейковый сервер Telegram —
// так же, как при запуске с AUTH_TELEGRAM_API_URL.
func TestListChangesThroughFakeTelegramAPI(t *testing.T) {
	type request struct {
		ChatID      int64  `json:"chat_id"`
		Text        string `json:"text"`
		ParseMode   string `json:"parse_mode"`
		ReplyMarkup struct {
			InlineKeyboard [][]core_telegram.Button `json:"inline_keyboard"`
		} `json:"reply_markup"`
	}
	var (
		got  []request
		path string
	)
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		var req request
		_ = json.NewDecoder(r.Body).Decode(&req)
		got = append(got, req)
		_, _ = w.Write([]byte(`{"ok":true,"result":{}}`))
	}))
	defer api.Close()

	repo := &fakeRepo{claimed: map[int]bool{}, digestDays: map[int]string{}, batches: []ListChangeBatch{homeBatch()}}
	n := NewNotifier(repo, core_telegram.NewClient(api.URL, "123:token"), "todo_bot", core_logger.FromContext(context.Background()))
	n.now = func() time.Time { return at(5, 10, 5) }

	n.Tick(context.Background())

	if len(got) != 1 {
		t.Fatalf("telegram got %d requests, want 1", len(got))
	}
	if path != "/bot123:token/sendMessage" {
		t.Fatalf("path = %q", path)
	}
	req := got[0]
	if req.ChatID != 77 || req.ParseMode != "HTML" || !strings.Contains(req.Text, "Купить молоко — Анна") {
		t.Fatalf("request = %+v", req)
	}
	if b := req.ReplyMarkup.InlineKeyboard[0][0]; b.Text != "Открыть список" || b.URL != "https://t.me/todo_bot?startapp=list_3" {
		t.Fatalf("button = %+v", b)
	}
	if !repo.sentChanges[1] || !repo.sentChanges[2] || !repo.sentChanges[3] {
		t.Fatalf("changes not marked sent: %v", repo.sentChanges)
	}
}
