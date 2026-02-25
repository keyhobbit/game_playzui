package repository

import (
	"context"
	"database/sql"

	"github.com/game-playzui/tienlen-server/internal/models"
)

type UserRepo struct {
	db *sql.DB
}

func NewUserRepo(db *sql.DB) *UserRepo {
	return &UserRepo{db: db}
}

func (r *UserRepo) Create(ctx context.Context, username, passwordHash string) (*models.User, error) {
	user := &models.User{}
	err := r.db.QueryRowContext(ctx,
		`INSERT INTO users (username, password_hash) VALUES ($1, $2)
		 RETURNING id, username, password_hash, gold_balance, rank, created_at`,
		username, passwordHash,
	).Scan(&user.ID, &user.Username, &user.PasswordHash, &user.GoldBalance, &user.Rank, &user.CreatedAt)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (r *UserRepo) FindByUsername(ctx context.Context, username string) (*models.User, error) {
	user := &models.User{}
	err := r.db.QueryRowContext(ctx,
		`SELECT id, username, password_hash, gold_balance, rank, created_at FROM users WHERE username = $1`,
		username,
	).Scan(&user.ID, &user.Username, &user.PasswordHash, &user.GoldBalance, &user.Rank, &user.CreatedAt)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (r *UserRepo) FindByID(ctx context.Context, id int64) (*models.User, error) {
	user := &models.User{}
	err := r.db.QueryRowContext(ctx,
		`SELECT id, username, password_hash, gold_balance, rank, created_at FROM users WHERE id = $1`,
		id,
	).Scan(&user.ID, &user.Username, &user.PasswordHash, &user.GoldBalance, &user.Rank, &user.CreatedAt)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (r *UserRepo) UpdateGold(ctx context.Context, userID int64, delta int64) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET gold_balance = gold_balance + $1 WHERE id = $2`,
		delta, userID,
	)
	return err
}
