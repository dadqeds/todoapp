package core_pgx_pool

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"time"

	core_postgres_pool "github.com/dadqeds/todoapp/internal/core/repository/postgres/pool"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Pool struct {
	*pgxpool.Pool
	opTimeout time.Duration
}

func connectionString(config Config) string {
	u := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(config.User, config.Password),
		Host:     net.JoinHostPort(config.Host, config.Port),
		Path:     config.Database,
		RawQuery: url.Values{"sslmode": {config.SSLMode}}.Encode(),
	}

	return u.String()
}

func NewPool(
	ctx context.Context,
	config Config,
) (*Pool, error) {
	pgxconfig, err := pgxpool.ParseConfig(connectionString(config))
	if err != nil {
		return nil, fmt.Errorf("parse pgxconfig: %w", err)
	}

	if config.MaxConns > 0 {
		pgxconfig.MaxConns = config.MaxConns
	}
	if config.MinConns > 0 {
		pgxconfig.MinConns = config.MinConns
	}
	if config.MaxConnLifetime > 0 {
		pgxconfig.MaxConnLifetime = config.MaxConnLifetime
	}
	if config.MaxConnIdleTime > 0 {
		pgxconfig.MaxConnIdleTime = config.MaxConnIdleTime
	}

	pool, err := pgxpool.NewWithConfig(ctx, pgxconfig)
	if err != nil {
		return nil, fmt.Errorf("create pgxpool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("pgxpool ping: %w", err)
	}
	return &Pool{
		Pool:      pool,
		opTimeout: config.Timeout,
	}, nil
}

func (p *Pool) Query(
	ctx context.Context,
	sql string,
	args ...any,
) (core_postgres_pool.Rows, error) {
	rows, err := p.Pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, mapErrors(err)
	}

	return pgxRows{rows}, nil
}

func (p *Pool) QueryRow(
	ctx context.Context,
	sql string,
	args ...any,
) core_postgres_pool.Row {
	row := p.Pool.QueryRow(ctx, sql, args...)

	return pgxRow{row}
}

func (p *Pool) Exec(
	ctx context.Context,
	sql string,
	arguments ...any,
) (core_postgres_pool.CommandTag, error) {
	tag, err := p.Pool.Exec(ctx, sql, arguments...)
	if err != nil {
		return nil, mapErrors(err)
	}
	return pgxCommandTag{tag}, nil
}

func (p *Pool) OpTimeout() time.Duration {
	return p.opTimeout
}
