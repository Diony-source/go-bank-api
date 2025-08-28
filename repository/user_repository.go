// file: repository/user_repository.go

package repository

import (
	"database/sql"
	"go-bank-api/logger"
	"go-bank-api/model"

	"github.com/sirupsen/logrus"
)

// IUserRepository defines the contract for user database operations.
type IUserRepository interface {
	CreateUser(user *model.User) error
	GetUserByEmail(email string) (*model.User, error)
	GetUserByID(id int) (*model.User, error) // <-- EKLENEN ARAYÜZ METODU
	GetAllUsers() ([]*model.User, error)
	UpdateUserRole(userID int, newRole string) error
}

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

// GetUserByID retrieves a user by their primary key ID.
func (r *UserRepository) GetUserByID(id int) (*model.User, error) {
	log := logger.Log.WithField("user_id", id)
	log.Info("Executing query to get user by ID")

	user := &model.User{}
	query := `SELECT id, username, email, password, role, created_at FROM users WHERE id=$1`
	err := r.DB.QueryRow(query, id).Scan(&user.ID, &user.Username, &user.Email, &user.Password, &user.Role, &user.CreatedAt)
	if err != nil {
		if err != sql.ErrNoRows {
			log.WithError(err).Error("Failed to execute get user by ID query")
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

// UpdateUserRole updates a user's role in the database.
func (r *UserRepository) UpdateUserRole(userID int, newRole string) error {
	log := logger.Log.WithFields(logrus.Fields{
		"user_id":  userID,
		"new_role": newRole,
	})
	log.Info("Executing query to update user role")

	query := `UPDATE users SET role = $1 WHERE id = $2`
	result, err := r.DB.Exec(query, newRole, userID)
	if err != nil {
		log.WithError(err).Error("Failed to execute update user role query")
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.WithError(err).Error("Failed to get rows affected after role update")
		return err
	}

	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	log.Info("User role updated successfully")
	return nil
}
