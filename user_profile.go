package sockchat

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/kacperf531/sockchat/api"
	"github.com/kacperf531/sockchat/storage"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
)

type ProfileService struct {
	Store storage.UserStore
	Cache *redis.Client
}

func (s *ProfileService) Create(ctx context.Context, u *api.CreateProfileRequest) error {
	if u.Password == "" {
		return api.ErrPasswordRequired
	}
	if u.Nick == "" {
		return api.ErrNickRequired
	}
	pwHash, err := bcrypt.GenerateFromPassword([]byte(u.Password), 10)
	if err != nil {
		log.Printf("error occured during generating password hash: %v", err)
		return api.ErrInternal
	}
	userEntry := storage.User{Nick: u.Nick, PwHash: string(pwHash), Description: u.Description}

	err = s.Store.InsertUser(ctx, &userEntry)
	if err != nil {
		if err == api.ErrNickAlreadyUsed {
			return err
		}
		log.Printf("error adding new user to db: %v", err)
		return api.ErrInternal
	}
	return nil
}

func (s *ProfileService) Edit(ctx context.Context, nick string, req *api.EditProfileRequest) error {
	userEntry := api.PublicProfile{Nick: nick, Description: req.Description}

	err := s.Store.UpdatePublicProfile(ctx, &userEntry)
	if err != nil {
		log.Printf("error updating user in db: %v", err)
		return api.ErrInternal
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

func (s *ProfileService) GetProfile(ctx context.Context, nick string) (*api.PublicProfile, error) {
	if nick == "" {
		return nil, api.ErrNickRequired
	}
	userData, err := s.getUserData(ctx, nick)
	if err != nil {
		if err == api.ErrUserNotFound {
			return nil, err
		}
		return nil, api.ErrInternal
	}
	return &api.PublicProfile{Nick: userData.Nick, Description: userData.Description}, nil
}

func (s *ProfileService) getUserData(ctx context.Context, nick string) (*storage.User, error) {
	userData, err := s.getFromCache(ctx, nick)
	if err == redis.Nil {
		userData, err := s.Store.SelectUser(ctx, nick)
		if err != nil {
			return nil, api.ErrUserNotFound
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
	s.Cache.Set(ctx, u.Nick, v, 10*time.Second)
}

func (s *ProfileService) removeFromCache(ctx context.Context, nick string) {
	s.Cache.Del(ctx, nick)
}

func (s *ProfileService) getFromCache(ctx context.Context, nick string) (*storage.User, error) {
	userData, err := s.Cache.Get(ctx, nick).Result()
	if err != nil {
		return nil, err
	}
	var u storage.User
	if err := json.Unmarshal([]byte(userData), &u); err != nil {
		log.Print("warning: error unmarshaling user data from cache")
	}
	return &u, nil
}
