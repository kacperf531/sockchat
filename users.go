package sockchat

import (
	"context"
	"fmt"
	"log"

	"github.com/kacperf531/sockchat/errors"
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
	pwHash, err := bcrypt.GenerateFromPassword([]byte(u.Password), 10)
	if err != nil {
		log.Printf("error occured during generating password hash: %v", err)
		return fmt.Errorf(CouldNotCreateUserMsg)
	}
	userEntry := storage.User{Nick: u.Nick, PwHash: string(pwHash), Description: u.Description}

	err = s.store.InsertUser(ctx, &userEntry)
	if err != nil {
		if err == errors.ResourceConflict {
			return err
		}
		log.Printf("error adding new user to db: %v", err)
		return fmt.Errorf(CouldNotCreateUserMsg)
	}
	return nil
}

func (s *UserService) EditUser(ctx context.Context, u *UserRequest) error {
	if !s.isAuthValid(ctx, u.Nick, u.Password) {
		return errors.Unauthorized
	}

	userEntry := storage.User{Nick: u.Nick, Description: u.Description}

	err := s.store.UpdateUser(ctx, &userEntry)
	if err != nil {
		log.Printf("error updating user in db: %v", err)
		return fmt.Errorf(CouldNotUpdateUserMsg)
	}
	return nil
}

func (s *UserService) LoginUser(ctx context.Context, nick, password string) error {
	if !s.isAuthValid(ctx, nick, password) {
		return errors.Unauthorized
	}

	return nil
}

func (s *UserService) isAuthValid(ctx context.Context, nick, password string) bool {
	if password == "" {
		return false
	}
	if nick == "" {
		return false
	}
	userData, err := s.store.SelectUser(ctx, nick)
	if err != nil {
		return false
	}
	if err := bcrypt.CompareHashAndPassword([]byte(userData.PwHash), []byte(password)); err != nil {
		return false
	}
	return true
}
