package service

import (
	"errors"
	"strings"

	"github.com/skylink/skylink/internal/store"
)

type CFAccountService struct {
	store *store.Store
}

func NewCFAccountService(st *store.Store) *CFAccountService {
	return &CFAccountService{store: st}
}

func (s *CFAccountService) GetActivatableAccount(id int64) (*store.CFAccount, error) {
	if id <= 0 {
		return nil, errors.New("invalid id")
	}
	acc, err := s.store.GetCFAccount(id)
	if err != nil {
		return nil, err
	}
	if acc == nil || strings.TrimSpace(acc.APIToken) == "" {
		return nil, errors.New("account not found or token empty")
	}
	return acc, nil
}
