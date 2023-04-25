package sockchat

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/kacperf531/sockchat/common"
	"github.com/kacperf531/sockchat/storage"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
)

const (
	CouldNotCreateUserMsg = "sorry, an error occured and your account could not be created"
	CouldNotUpdateUserMsg = "sorry, an error occured and your account could not be updated"
)

type ProfileService struct {
	store storage.UserStore
	cache *redis.Client
}

func (s *ProfileService) Create(ctx context.Context, u *CreateProfileRequest) error {
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
		if err == common.ErrResourceConflict {
			return err
		}
		log.Printf("error adding new user to db: %v", err)
		return fmt.Errorf(CouldNotCreateUserMsg)
	}
	return nil
}

func (s *ProfileService) Edit(ctx context.Context, nick string, req *EditProfileRequest) error {
	userEntry := common.PublicProfile{Nick: nick, Description: req.Description}

	err := s.store.UpdatePublicProfile(ctx, &userEntry)
	if err != nil {
		log.Printf("error updating user in db: %v", err)
		return fmt.Errorf(CouldNotUpdateUserMsg)
	}
	s.removeFromCache(ctx, nick)
	return nil
}

func (s *ProfileService) IsAuthValid(ctx context.Context, nick, password string) bool {
	if nick == "" || password == "" {
		return false
	}
	userData, err := s.getUserData(ctx, nick)
	if err != nil {
		return false
	}
	if err := bcrypt.CompareHashAndPassword([]byte(userData.PwHash), []byte(password)); err != nil {
		return false
	}
	return true
}

func (s *ProfileService) GetProfile(ctx context.Context, nick string) (*common.PublicProfile, error) {
	userData, err := s.getUserData(ctx, nick)
	if err != nil {
		return nil, err
	}
	return &common.PublicProfile{Nick: userData.Nick, Description: userData.Description}, nil
}

func (s *ProfileService) getUserData(ctx context.Context, nick string) (*storage.User, error) {
	userData, err := s.getFromCache(ctx, nick)
	if err == redis.Nil {
		userData, err := s.store.SelectUser(ctx, nick)
		if err != nil {
			return nil, err
		}
		go s.setInCache(ctx, userData)
		return userData, nil
	}
	if err != nil {
		return nil, err
	}
	return userData, nil
}

func (s *ProfileService) setInCache(ctx context.Context, u *storage.User) {
	v, err := json.Marshal(u)
	if err != nil {
		log.Print("warning: error marshaling user data for cache")
	}
	s.cache.Set(ctx, u.Nick, v, 10*time.Second)
}

func (s *ProfileService) removeFromCache(ctx context.Context, nick string) {
	s.cache.Del(ctx, nick)
}

func (s *ProfileService) getFromCache(ctx context.Context, nick string) (*storage.User, error) {
	userData, err := s.cache.Get(ctx, nick).Result()
	if err != nil {
		return nil, err
	}
	var u storage.User
	if err := json.Unmarshal([]byte(userData), &u); err != nil {
		log.Print("warning: error unmarshaling user data from cache")
	}
	return &u, nil
}
