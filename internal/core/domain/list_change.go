package domain

// ListChangeKind — что произошло в общем списке. Отметки пунктов чеклиста
// и правки задач событиями не считаются: о них не сообщаем.
type ListChangeKind string

const (
	ListChangeTaskAdded     ListChangeKind = "task_added"
	ListChangeTaskCompleted ListChangeKind = "task_completed"
)

// ListChange — событие для уведомления остальных участников списка.
type ListChange struct {
	ListID      int
	ActorUserID int
	Kind        ListChangeKind
	TaskTitle   string
}
