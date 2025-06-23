package repository

import (
	"database/sql"
	"go-bank-api/model"
)

type UserRepository struct {
	DB *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{DB: db}
}

func (r *UserRepository) CreateUser(user *model.User) error {
	query := `INSERT INTO users (username, email, password) VALUES ($1, $2, $3) RETURNING id, created_at`
	return r.DB.QueryRow(query, user.Username, user.Email, user.Password).Scan(&user.ID, &user.CreatedAt)
}

func (r *UserRepository) GetUserByEmail(email string) (*model.User, error) {
	user := &model.User{}
	query := `SELECT id, username, email, password, created_at FROM users WHERE email=$1`
	err := r.DB.QueryRow(query, email).Scan(&user.ID, &user.Username, &user.Email, &user.Password, &user.CreatedAt)
	if err != nil {
		return nil, err
	}
	return user, nil
}
