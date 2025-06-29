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

	query := `INSERT INTO users (username, email, password) VALUES ($1, $2, $3) RETURNING id, created_at, role`
	err := r.DB.QueryRow(query, user.Username, user.Email, user.Password).Scan(&user.ID, &user.CreatedAt, &user.Role)
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
	query := `SELECT id, username, email, password, role, created_at FROM users WHERE email=$1`
	err := r.DB.QueryRow(query, email).Scan(&user.ID, &user.Username, &user.Email, &user.Password, &user.Role, &user.CreatedAt)
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

// GetAllUsers retrieves all users from the database. For admin use only.
func (r *UserRepository) GetAllUsers() ([]*model.User, error) {
	log := logger.Log
	log.Info("Executing query to get all users")

	query := `SELECT id, username, email, role, created_at FROM users`
	rows, err := r.DB.Query(query)
	if err != nil {
		log.WithError(err).Error("Failed to execute query for all users")
		return nil, err
	}
	defer rows.Close()

	var users []*model.User
	for rows.Next() {
		var user model.User
		if err := rows.Scan(&user.ID, &user.Username, &user.Email, &user.Role, &user.CreatedAt); err != nil {
			log.WithError(err).Error("Failed to scan user row")
			return nil, err
		}
		users = append(users, &user)
	}
	return users, nil
}