package repository

import (
	"database/sql"
	"go-bank-api/logger"
	"go-bank-api/model"

	"github.com/sirupsen/logrus"
)

type UserRepository struct {
	DB *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{DB: db}
}

func (r *UserRepository) CreateUser(user *model.User) error {
	log := logger.Log.WithFields(logrus.Fields{
		"username": user.Username,
		"email":    user.Email,
	})
	log.Info("Executing query to create a new user")

	query := `INSERT INTO users (username, email, password) VALUES ($1, $2, $3) RETURNING id, created_at`
	err := r.DB.QueryRow(query, user.Username, user.Email, user.Password).Scan(&user.ID, &user.CreatedAt)
	if err != nil {
		log.WithError(err).Error("Failed to execute create user query")
		return err
	}
	return nil
}

func (r *UserRepository) GetUserByEmail(email string) (*model.User, error) {
	log := logger.Log.WithField("email", email)
	log.Info("Executing query to get user by email")

	user := &model.User{}
	query := `SELECT id, username, email, password, created_at FROM users WHERE email=$1`
	err := r.DB.QueryRow(query, email).Scan(&user.ID, &user.Username, &user.Email, &user.Password, &user.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Info("User not found in database")
		} else {
			log.WithError(err).Error("Failed to execute get user by email query")
		}
		return nil, err
	}
	return user, nil
}
