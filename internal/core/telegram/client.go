// Package core_telegram — минимальный клиент Telegram Bot API для отправки сообщений.
package core_telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var (
	// ErrChatUnavailable — писать этому человеку нельзя: он не запускал бота,
	// заблокировал его или не дал разрешения. Повторять бессмысленно.
	ErrChatUnavailable = errors.New("telegram chat unavailable")
)

const DefaultAPIURL = "https://api.telegram.org"

type Client struct {
	apiURL string
	token  string
	http   *http.Client
}

func NewClient(apiURL string, token string) *Client {
	if apiURL == "" {
		apiURL = DefaultAPIURL
	}
	return &Client{
		apiURL: strings.TrimRight(apiURL, "/"),
		token:  token,
		http:   &http.Client{Timeout: 15 * time.Second},
	}
}

// Button — кнопка-ссылка под сообщением.
type Button struct {
	Text string `json:"text"`
	URL  string `json:"url"`
}

type sendMessageRequest struct {
	ChatID      int64        `json:"chat_id"`
	Text        string       `json:"text"`
	ParseMode   string       `json:"parse_mode"`
	ReplyMarkup *replyMarkup `json:"reply_markup,omitempty"`

	LinkPreviewOptions struct {
		IsDisabled bool `json:"is_disabled"`
	} `json:"link_preview_options"`
}

type replyMarkup struct {
	InlineKeyboard [][]Button `json:"inline_keyboard"`
}

type apiResponse struct {
	OK          bool   `json:"ok"`
	ErrorCode   int    `json:"error_code"`
	Description string `json:"description"`
}

// SendMessage отправляет HTML-сообщение с необязательной кнопкой.
func (c *Client) SendMessage(ctx context.Context, chatID int64, html string, button *Button) error {
	req := sendMessageRequest{ChatID: chatID, Text: html, ParseMode: "HTML"}
	req.LinkPreviewOptions.IsDisabled = true
	if button != nil {
		req.ReplyMarkup = &replyMarkup{InlineKeyboard: [][]Button{{*button}}}
	}

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.apiURL+"/bot"+c.token+"/sendMessage", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build request: %s", redact(err))
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(httpReq)
	if err != nil {
		// В адресе запроса есть токен бота: в ошибку он попасть не должен.
		return fmt.Errorf("send message: %s", redact(err))
	}
	defer func() { _ = resp.Body.Close() }()

	var result apiResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("decode telegram response (HTTP %d): %w", resp.StatusCode, err)
	}

	if !result.OK {
		if result.ErrorCode == http.StatusForbidden ||
			(result.ErrorCode == http.StatusBadRequest && strings.Contains(result.Description, "chat not found")) {
			return fmt.Errorf("%s: %w", result.Description, ErrChatUnavailable)
		}
		return fmt.Errorf("telegram error %d: %s", result.ErrorCode, result.Description)
	}

	return nil
}

// redact убирает из сетевой ошибки адрес запроса вместе с токеном.
func redact(err error) string {
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		return urlErr.Op + ": " + urlErr.Err.Error()
	}
	return err.Error()
}
