package storage

import (
	"context"
	"database/sql"
	"fmt"
	"os"

	"github.com/VividCortex/mysqlerr"
	"github.com/go-sql-driver/mysql"
	_ "github.com/go-sql-driver/mysql"
	"github.com/kacperf531/sockchat/common"
)

type UserStore interface {
	InsertUser(context.Context, *User) error
	UpdateUser(context.Context, *User) error
	SelectUser(context.Context, string) (*User, error)
}

func NewUserStore(db *sql.DB) UserStore {
	return &userStore{
		db: db,
	}
}

type userStore struct {
	db *sql.DB
}

type User struct {
	Nick        string
	PwHash      string
	Description string
}

func (s *userStore) InsertUser(ctx context.Context, u *User) error {
	const stmt = "INSERT INTO users(nick, pw_hash, description) VALUES (?, ?, ?);  "

	res, err := s.db.ExecContext(ctx, stmt, u.Nick, u.PwHash, u.Description)
	if err != nil {
		if driverErr, ok := err.(*mysql.MySQLError); ok {
			if driverErr.Number == mysqlerr.ER_DUP_ENTRY {
				return common.ErrResourceConflict
			}
		}
		return fmt.Errorf("could not insert row: %w", err)
	}

	if _, err := res.RowsAffected(); err != nil {
		return fmt.Errorf("could not get affected rows: %w", err)
	}

	return nil
}

func (s *userStore) UpdateUser(ctx context.Context, u *User) error {
	const stmt = "UPDATE users SET description = ? WHERE nick = ?;  "

	res, err := s.db.ExecContext(ctx, stmt, u.Description, u.Nick)
	if err != nil {
		return fmt.Errorf("could not update row: %w", err)
	}

	if _, err := res.RowsAffected(); err != nil {
		return fmt.Errorf("could not get affected rows: %w", err)
	}

	return nil
}

func (s *userStore) SelectUser(ctx context.Context, nick string) (*User, error) {
	var user User
	if err := s.db.QueryRow("SELECT nick, pw_hash, description FROM users WHERE nick = ?;", nick).Scan(&user.Nick, &user.PwHash, &user.Description); err != nil {
		return nil, fmt.Errorf("could not get row: %w", err)
	}

	return &user, nil
}

func ResetUsersTable(db *sql.DB) error {
	db.Exec("DROP TABLE IF EXISTS users;")
	commandBytes, err := os.ReadFile("../storage/create-users.sql")
	if err != nil {
		return fmt.Errorf("could not load script for creating users")
	}
	_, err = db.Exec(string(commandBytes))
	return err
}
