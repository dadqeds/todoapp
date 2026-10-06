package notifications_service

import (
	"context"
	"errors"
	"fmt"
	"html"
	"strings"
	"time"

	"github.com/dadqeds/todoapp/internal/core/domain"
	core_telegram "github.com/dadqeds/todoapp/internal/core/telegram"
	"go.uber.org/zap"
)

const (
	// Изменения копятся столько с первого события, затем уходят одним сообщением.
	listChangesDelay = 3 * time.Minute
	// Сколько хранить очередь: отправленное и то, что отправить не удалось.
	listChangesKeep = 24 * time.Hour
	// Сколько задач показывать в одном разделе сообщения.
	listChangesShown = 10
)

func (n *Notifier) sendListChanges(ctx context.Context, now time.Time) {
	batches, err := n.repo.GetPendingListChanges(ctx, now.Add(-listChangesDelay))
	if err != nil {
		n.log.Error("get pending list changes", zap.Error(err))
		return
	}

	for _, b := range batches {
		n.processListChanges(ctx, b)
	}

	if err := n.repo.DeleteListChangesBefore(ctx, now.Add(-listChangesKeep)); err != nil {
		n.log.Error("delete old list changes", zap.Error(err))
	}
}

func (n *Notifier) processListChanges(ctx context.Context, b ListChangeBatch) {
	ids := make([]int64, len(b.Changes))
	for i, c := range b.Changes {
		ids[i] = c.ID
	}
	log := n.log.With(zap.Int("list_id", b.ListID), zap.Int("user_id", b.RecipientUserID))

	claimed, err := n.repo.ClaimListChanges(ctx, ids)
	if err != nil {
		log.Error("claim list changes", zap.Error(err))
		return
	}
	if !claimed {
		return
	}

	err = n.sender.SendMessage(ctx, b.ChatID, listChangesText(b), n.openButton("Открыть список", fmt.Sprintf("list_%d", b.ListID)))
	switch {
	case err == nil:
		log.Info("list changes sent", zap.Int("changes", len(ids)))
	case errors.Is(err, core_telegram.ErrChatUnavailable):
		// Человек не разрешил боту писать — повторять бессмысленно.
		log.Warn("list changes not delivered", zap.Error(err))
	default:
		log.Warn("send list changes, will retry", zap.Error(err))
		if err := n.repo.ReleaseListChanges(ctx, ids); err != nil {
			log.Error("release list changes", zap.Error(err))
		}
	}
}

// listChangesText — что добавили и что выполнили, с именами, кто это сделал.
func listChangesText(b ListChangeBatch) string {
	var added, completed []string
	for _, c := range b.Changes {
		line := "• " + html.EscapeString(c.TaskTitle) + " — " + html.EscapeString(c.ActorName)
		switch domain.ListChangeKind(c.Kind) {
		case domain.ListChangeTaskAdded:
			added = append(added, line)
		case domain.ListChangeTaskCompleted:
			completed = append(completed, line)
		}
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "<b>Изменения в списке «%s»</b>", html.EscapeString(b.ListTitle))
	section := func(title string, lines []string) {
		if len(lines) == 0 {
			return
		}
		fmt.Fprintf(&sb, "\n\n<b>%s</b>", title)
		for i, line := range lines {
			if i == listChangesShown {
				rest := len(lines) - listChangesShown
				fmt.Fprintf(&sb, "\n…и ещё %d %s", rest, plural(rest, "задача", "задачи", "задач"))
				break
			}
			sb.WriteString("\n" + line)
		}
	}
	section("Добавлены", added)
	section("Выполнены", completed)

	return sb.String()
}
