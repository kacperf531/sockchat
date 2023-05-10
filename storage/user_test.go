package storage

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"github.com/joho/godotenv"
	"github.com/kacperf531/sockchat/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserStore(t *testing.T) {
	godotenv.Load("../.env")

	db := mustSetUpTestDB(t)
	defer db.Close()

	store := NewUserStore(db)
	var userExists bool

	t.Run("inserts new user into DB", func(t *testing.T) {
		createUserFoo(t, store)
		userExists = true
	})

	t.Run("updates existing user's info in DB", func(t *testing.T) {
		if !userExists {
			createUserFoo(t, store)
			userExists = true
		}
		err := store.UpdatePublicProfile(context.TODO(), &api.PublicProfile{
			Nick:        "Foo",
			Description: "Baz",
		})
		require.NoError(t, err)
	})

	t.Run("returns existing user's info in DB", func(t *testing.T) {
		if !userExists {
			createUserFoo(t, store)
			userExists = true
		}
		user, err := store.SelectUser(context.TODO(), "Foo")
		require.NoError(t, err)
		assert.Equal(t, "Foo", user.Nick)
	})

}

func mustSetUpTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("mysql", os.Getenv("DB_USER")+":"+os.Getenv("DB_PASSWORD")+"@tcp("+os.Getenv("DB_HOST")+")/sockchat_test")
	if err != nil {
		t.Errorf("could not connect to the DB due to an error: %v", err)
	}

	err = ResetUsersTable(db)
	if err != nil {
		t.Errorf("error setting up the table %v", err)
	}
	return db
}

func createUserFoo(t *testing.T, store UserStore) *User {
	t.Helper()
	newUser := &User{
		Nick:        "Foo",
		PwHash:      "Bar",
		Description: "desc"}
	err := store.InsertUser(context.TODO(), newUser)
	if err != nil {
		t.Errorf("could not insert new user due to an error %v", err)
	}
	return newUser
}
