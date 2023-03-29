package storage

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql"
)

type UserStore interface {
	InsertUser(context.Context, *User) error
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
	Nick   string
	PwHash string
	Salt   string
}

func (s *userStore) InsertUser(ctx context.Context, u *User) error {
	const stmt = "INSERT INTO users(Nick, PwHash, Salt) VALUES (?, ?, ?);  "

	res, err := s.db.ExecContext(ctx, stmt, u.Nick, u.PwHash, u.Salt)
	if err != nil {
		return fmt.Errorf("could not insert row: %w", err)
	}

	if _, err := res.RowsAffected(); err != nil {
		return fmt.Errorf("could not get affected rows: %w", err)
	}

	return nil
}
