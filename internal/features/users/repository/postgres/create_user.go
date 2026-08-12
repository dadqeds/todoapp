package users_postgres_repository

import (
	"context"
	"fmt"

	"github.com/dadqeds/todoapp/internal/core/domain"
)

func (r *UsersRepository)  CreateUser(
	ctx context.Context,
	user domain.User,
	)(domain.User, error) {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OpTimeout())
	defer cancel()

	query := `
	INSERT INTO todoapp.users (full_name, phone_number)
	VALUES ($1, $2)
	RETURNING id, version, full_name, phone_number;
	`
	row := r.pool.QueryRow(ctx,query,user.FullName, user.PhoneNumber)
	var UserModel UserModel
	err := row.Scan(
		&UserModel.ID,
		&UserModel.Version,
		&UserModel.FullName,
		&UserModel.PhoneNumber,
	)
	if err != nil{
		return domain.User{}, fmt.Errorf("scan error: %w", err)
	}
	userDomain := domain.NewUser(
		UserModel.ID,
		UserModel.Version,
		UserModel.FullName,
		UserModel.PhoneNumber,
	)

	return userDomain, nil
}