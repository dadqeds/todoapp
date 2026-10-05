package tasks_transport_http

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dadqeds/todoapp/internal/core/domain"
	core_errors "github.com/dadqeds/todoapp/internal/core/errors"
)

type fakeTasksService struct {
	TasksService
	deleteErr error
}

func (f fakeTasksService) DeleteTask(context.Context, int) error { return f.deleteErr }

func (f fakeTasksService) GetTasks(context.Context, *int, *int, *int) ([]domain.Task, error) {
	return nil, nil
}

func serve(h http.HandlerFunc, method, pattern, target string) *httptest.ResponseRecorder {
	mux := http.NewServeMux()
	mux.HandleFunc(method+" "+pattern, h)

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(method, target, nil))
	return rec
}

func TestDeleteTaskNotFoundWritesSingleResponse(t *testing.T) {
	h := NewTasksHTTPHandler(fakeTasksService{
		deleteErr: fmt.Errorf("task 1: %w", core_errors.ErrNotFound),
	})

	rec := serve(h.DeleteTask, http.MethodDelete, "/tasks/{id}", "/tasks/1")

	if rec.Code != http.StatusNotFound {
		t.Fatalf("code = %d, want 404", rec.Code)
	}
}

func TestDeleteTaskOK(t *testing.T) {
	h := NewTasksHTTPHandler(fakeTasksService{})

	rec := serve(h.DeleteTask, http.MethodDelete, "/tasks/{id}", "/tasks/1")

	if rec.Code != http.StatusNoContent || rec.Body.Len() != 0 {
		t.Fatalf("code = %d, body = %q", rec.Code, rec.Body.String())
	}
}

func TestGetTasksReturnsEmptyArray(t *testing.T) {
	h := NewTasksHTTPHandler(fakeTasksService{})

	rec := serve(h.GetTasks, http.MethodGet, "/tasks", "/tasks")

	if rec.Code != http.StatusOK || rec.Body.String() != "[]\n" {
		t.Fatalf("code = %d, body = %q", rec.Code, rec.Body.String())
	}
}
