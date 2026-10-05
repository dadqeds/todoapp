package core_telegram

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const token = "123:SECRET-token"

func TestSendMessage(t *testing.T) {
	var got sendMessageRequest
	var path string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&got)
		_, _ = w.Write([]byte(`{"ok":true,"result":{}}`))
	}))
	defer srv.Close()

	err := NewClient(srv.URL, token).SendMessage(context.Background(), 42, "<b>hi</b>", &Button{Text: "Открыть", URL: "https://t.me/bot?startapp"})
	if err != nil {
		t.Fatal(err)
	}
	if path != "/bot"+token+"/sendMessage" {
		t.Fatalf("path = %q", path)
	}
	if got.ChatID != 42 || got.ParseMode != "HTML" || got.ReplyMarkup.InlineKeyboard[0][0].Text != "Открыть" {
		t.Fatalf("request = %+v", got)
	}
}

func TestSendMessageErrors(t *testing.T) {
	respond := func(body string) *httptest.Server {
		return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(body))
		}))
	}

	blocked := respond(`{"ok":false,"error_code":403,"description":"Forbidden: bot was blocked by the user"}`)
	defer blocked.Close()
	if err := NewClient(blocked.URL, token).SendMessage(context.Background(), 1, "x", nil); !errors.Is(err, ErrChatUnavailable) {
		t.Errorf("403: err = %v, want ErrChatUnavailable", err)
	}

	limited := respond(`{"ok":false,"error_code":429,"description":"Too Many Requests"}`)
	defer limited.Close()
	if err := NewClient(limited.URL, token).SendMessage(context.Background(), 1, "x", nil); err == nil || errors.Is(err, ErrChatUnavailable) {
		t.Errorf("429: err = %v, want transient error", err)
	}
}

func TestNetworkErrorDoesNotLeakToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	url := srv.URL
	srv.Close()

	err := NewClient(url, token).SendMessage(context.Background(), 1, "x", nil)
	if err == nil {
		t.Fatal("want error")
	}
	if strings.Contains(err.Error(), "SECRET") {
		t.Fatalf("token leaked into error: %v", err)
	}
}
