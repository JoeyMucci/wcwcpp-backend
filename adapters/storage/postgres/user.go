package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/go-jet/jet/v2/postgres"
	"github.com/go-jet/jet/v2/qrm"
	"github.com/joey/wcwcpp-backend/adapters/storage/jet/wcwcpp/public/model"
	"github.com/joey/wcwcpp-backend/adapters/storage/jet/wcwcpp/public/table"
	"github.com/joey/wcwcpp-backend/core/entity"
)

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*entity.User, error) {
	stmt := postgres.SELECT(table.Users.AllColumns).
		FROM(table.Users).
		WHERE(table.Users.Email.EQ(postgres.String(email)))

	var dest model.Users
	err := stmt.QueryContext(ctx, r.db, &dest)
	if err != nil {
		if errors.Is(err, qrm.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return &entity.User{
		ID:       dest.ID.String(),
		Email:    dest.Email,
		Username: dest.Username,
	}, nil
}

func (r *UserRepository) CreateUser(ctx context.Context, email string, username string) (*entity.User, error) {
	stmt := table.Users.INSERT(table.Users.Email, table.Users.Username).
		VALUES(email, username).
		RETURNING(table.Users.AllColumns)

	var dest model.Users
	err := stmt.QueryContext(ctx, r.db, &dest)
	if err != nil {
		return nil, err
	}

	return &entity.User{
		ID:       dest.ID.String(),
		Email:    dest.Email,
		Username: dest.Username,
	}, nil
}
