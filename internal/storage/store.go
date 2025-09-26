package storage

import (
	"errors"
	"sync"

	"github.com/yonatannn111/Online_Polls-Voting_System_in_Go/internal/models"
)

type Store struct {
	mu    sync.Mutex
	polls map[string]*models.Poll
}

// NewStore initializes storage
func NewStore() *Store {
	return &Store{polls: make(map[string]*models.Poll)}
}

// CreatePoll adds a new poll
func (s *Store) CreatePoll(poll *models.Poll) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.polls[poll.ID] = poll
}

// GetPoll returns a poll by ID
func (s *Store) GetPoll(id string) (*models.Poll, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	poll, exists := s.polls[id]
	if !exists {
		return nil, errors.New("poll not found")
	}
	return poll, nil
}

// Vote adds a vote to a poll
func (s *Store) Vote(id string, option string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	poll, exists := s.polls[id]
	if !exists {
		return errors.New("poll not found")
	}

	_, ok := poll.Votes[option]
	if !ok {
		return errors.New("invalid option")
	}
	poll.Votes[option]++
	return nil
}

// ListPolls returns all polls
func (s *Store) ListPolls() []*models.Poll {
	s.mu.Lock()
	defer s.mu.Unlock()
	list := []*models.Poll{}
	for _, p := range s.polls {
		list = append(list, p)
	}
	return list
}
