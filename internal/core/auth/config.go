package core_auth

import (
	"fmt"
	"slices"
	"time"

	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	// Токен бота от @BotFather. Без него вход через Telegram невозможен:
	// публичный API отвечает 401, работает только локальный адрес.
	TelegramBotToken string `envconfig:"TELEGRAM_BOT_TOKEN"`
	// Имя бота без @ — для ссылок-приглашений t.me/<бот>?startapp=...
	TelegramBotUsername string `envconfig:"TELEGRAM_BOT_USERNAME"`
	// Адрес Bot API; меняется только для проверки с заглушкой.
	TelegramAPIURL   string        `envconfig:"TELEGRAM_API_URL" default:"https://api.telegram.org"`
	InitDataMaxAge   time.Duration `envconfig:"INIT_DATA_MAX_AGE"   default:"24h"`
	AdminTelegramIDs []int64       `envconfig:"ADMIN_TELEGRAM_IDS"`

	// Локальный адрес без проверки Telegram. Слушать его нужно только на
	// 127.0.0.1 и никогда не пробрасывать наружу (туннель смотрит на HTTP_ADDR).
	LocalAddr       string `envconfig:"LOCAL_ADDR"`
	LocalTelegramID int64  `envconfig:"LOCAL_TELEGRAM_ID"`
}

func NewConfig() (Config, error) {
	var config Config

	if err := envconfig.Process("AUTH", &config); err != nil {
		return Config{}, fmt.Errorf("process envconfig: %w", err)
	}

	if config.LocalAddr != "" && config.LocalTelegramID == 0 {
		return Config{}, fmt.Errorf("AUTH_LOCAL_TELEGRAM_ID is required when AUTH_LOCAL_ADDR is set")
	}

	return config, nil
}

func NewConfigMust() Config {
	config, err := NewConfig()
	if err != nil {
		panic(fmt.Errorf("get auth config: %w", err))
	}
	return config
}

func (c Config) IsAdmin(telegramID int64) bool {
	return slices.Contains(c.AdminTelegramIDs, telegramID)
}
