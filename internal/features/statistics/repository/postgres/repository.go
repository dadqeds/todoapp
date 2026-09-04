package statistics_postgres_repository

import core_postgres_pool "github.com/dadqeds/todoapp/internal/core/repository/postgres/pool"

type StatisticsRepository struct {
	poll core_postgres_pool.Pool
}

func NewStatisticsRepository(
	poll core_postgres_pool.Pool,
) *StatisticsRepository {
	return &StatisticsRepository{
		poll: poll,
	}
}
