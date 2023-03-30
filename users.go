package sockchat

import (
	"context"
	"fmt"
	"log"

	"github.com/kacperf531/sockchat/storage"
	"golang.org/x/crypto/bcrypt"
)

const (
	CouldNotCreateUserMsg = "sorry, an error occured and your account could not be created"
	CouldNotUpdateUserMsg = "sorry, an error occured and your account could not be updated"
)

type UserService struct {
	store storage.UserStore
}

func (s *UserService) CreateUser(ctx context.Context, u *UserRequest) error {
	if u.Password == "" {
		return fmt.Errorf("password is required")
	}
	if u.Nick == "" {
		return fmt.Errorf("you must provide your nick")
	}
	if err, _ := s.getUser(ctx, u.Nick); err != nil {
		return fmt.Errorf("this nick is already taken")
	}
	pwHash, err := bcrypt.GenerateFromPassword([]byte(u.Password), 10)
	if err != nil {
		log.Printf("error occured during generating password hash: %v", err)
		return fmt.Errorf(CouldNotCreateUserMsg)
	}
	userEntry := storage.User{Nick: u.Nick, PwHash: string(pwHash), Description: u.Description}

	err = s.store.InsertUser(ctx, &userEntry)
	if err != nil {
		log.Printf("error adding new user to db: %v", err)
		return fmt.Errorf(CouldNotCreateUserMsg)
	}
	return nil
}

func (s *UserService) EditUser(ctx context.Context, u *UserRequest) error {
	if u.Password == "" {
		return fmt.Errorf("authorization with password is required to edit user")
	}
	userData, err := s.getUser(ctx, u.Nick)
	if err != nil {
		return fmt.Errorf("invalid authorization")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(userData.PwHash), []byte(u.Password)); err != nil {
		return fmt.Errorf("invalid authorization")
	}
	if u.Nick == "" {
		return fmt.Errorf("you must provide your nick")
	}
	userEntry := storage.User{Nick: u.Nick, Description: u.Description}

	err = s.store.UpdateUser(ctx, &userEntry)
	if err != nil {
		log.Printf("error adding new user to db: %v", err)
		return fmt.Errorf(CouldNotUpdateUserMsg)
	}
	return nil
}

func (s *UserService) getUser(ctx context.Context, nick string) (*storage.User, error) {

	userEntry, err := s.store.SelectUser(ctx, nick)

	if err != nil {
		return nil, fmt.Errorf("could not find user %s", nick)
	}
	return userEntry, nil
}
