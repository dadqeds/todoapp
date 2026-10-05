package core_pgx_pool

import (
	"fmt"
	"time"

	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	Host     string        `envconfig:"HOST"     required:"true"`
	Port     string        `envconfig:"PORT"     default:"5432"`
	User     string        `envconfig:"USER"     required:"true"`
	Password string        `envconfig:"PASSWORD" required:"true"`
	Database string        `envconfig:"DB"       required:"true"`
	SSLMode  string        `envconfig:"SSLMODE"  default:"disable"`
	Timeout  time.Duration `envconfig:"TIMEOUT"  required:"true"`

	// Нулевые значения означают "оставить дефолт pgxpool".
	MaxConns        int32         `envconfig:"MAX_CONNS"`
	MinConns        int32         `envconfig:"MIN_CONNS"`
	MaxConnLifetime time.Duration `envconfig:"MAX_CONN_LIFETIME"`
	MaxConnIdleTime time.Duration `envconfig:"MAX_CONN_IDLE_TIME"`
}

func NewConfig() (Config, error) {
	var config Config

	if err := envconfig.Process("POSTGRES", &config); err != nil {
		return Config{}, fmt.Errorf("process envconfig: %w", err)
	}

	return config, nil
}

func NewConfigMust() Config {
	config, err := NewConfig()
	if err != nil {
		panic(fmt.Errorf("get postgres connection pool config: %w", err))
	}
	return config
}
