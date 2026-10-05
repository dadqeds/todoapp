package lists_postgres_repository

import core_postgres_pool "github.com/dadqeds/todoapp/internal/core/repository/postgres/pool"

type ListsRepository struct {
	pool core_postgres_pool.Pool
}

func NewListsRepository(pool core_postgres_pool.Pool) *ListsRepository {
	return &ListsRepository{
		pool: pool,
	}
}
