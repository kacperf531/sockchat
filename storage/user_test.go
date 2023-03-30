package storage

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/require"
)

func TestUserStore(t *testing.T) {
	godotenv.Load("../.env")

	db := mustSetUpTestDB(t)
	defer db.Close()

	store := NewUserStore(db)

	t.Run("inserts new user into DB", func(t *testing.T) {
		err := store.InsertUser(context.TODO(), &User{
			Nick:        "Foo",
			PwHash:      "Bar",
			Description: "desc",
		})
		require.NoError(t, err)
	})

	t.Run("updates existing user's info in DB", func(t *testing.T) {
		err := store.UpdateUser(context.TODO(), &User{
			Nick:        "Foo",
			Description: "Baz",
		})
		require.NoError(t, err)
	})

}

func mustSetUpTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("mysql", os.Getenv("DB_USER")+":"+os.Getenv("DB_PASSWORD")+"@tcp("+os.Getenv("DB_HOST")+")/sockchat_test")
	if err != nil {
		t.Errorf("could not connect to the DB due to an error: %v", err)
	}

	db.Exec("DROP TABLE IF EXISTS users;")
	_, err = db.Exec(`CREATE TABLE users (
		nick      VARCHAR(255) NOT NULL,
		pw_hash     VARCHAR(255) NOT NULL,
		description      VARCHAR(255) NOT NULL
	  );`)
	if err != nil {
		t.Errorf("error setting up the table %v", err)
	}
	return db
}
