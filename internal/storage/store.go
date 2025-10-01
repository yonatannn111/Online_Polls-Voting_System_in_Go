package storage

import (
	"context"
	"errors"
	"fmt"

	"cloud.google.com/go/firestore"
	"github.com/yonatannn111/Online_Polls-Voting_System_in_Go/internal/models"
)

type Store struct {
	client *firestore.Client
}

// NewStore initializes Firestore-backed store
func NewStore(client *firestore.Client) *Store {
	return &Store{client: client}
}

// CreatePoll adds a new poll to Firestore
func (s *Store) CreatePoll(poll *models.Poll) error {
	ctx := context.Background()
	docRef := s.client.Collection("polls").Doc(poll.ID)

	// Check if poll already exists
	doc, err := docRef.Get(ctx)
	if err == nil && doc.Exists() {
		return fmt.Errorf("poll with ID %s already exists", poll.ID)
	}

	_, err = docRef.Set(ctx, poll)
	return err
}

// GetPoll returns a poll by ID from Firestore
func (s *Store) GetPoll(id string) (*models.Poll, error) {
	ctx := context.Background()
	doc, err := s.client.Collection("polls").Doc(id).Get(ctx)
	if err != nil || !doc.Exists() {
		return nil, errors.New("poll not found")
	}
	var poll models.Poll
	if err := doc.DataTo(&poll); err != nil {
		return nil, err
	}
	return &poll, nil
}

// Vote adds a vote to a poll
func (s *Store) Vote(id string, option string) error {
	ctx := context.Background()
	docRef := s.client.Collection("polls").Doc(id)

	 err := s.client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		doc, err := tx.Get(docRef)
		if err != nil {
			return err
		}

		var poll models.Poll
		if err := doc.DataTo(&poll); err != nil {
			return err
		}

		if _, ok := poll.Votes[option]; !ok {
			return fmt.Errorf("invalid option")
		}

		poll.Votes[option]++
		return tx.Set(docRef, poll)
	})
	return err
}

// ListPolls returns all polls from Firestore
func (s *Store) ListPolls() []*models.Poll {
	ctx := context.Background()
	iter := s.client.Collection("polls").Documents(ctx)
	var polls []*models.Poll
	for {
		doc, err := iter.Next()
		if err != nil {
			break
		}
		var poll models.Poll
		if err := doc.DataTo(&poll); err == nil {
			polls = append(polls, &poll)
		}
	}
	return polls
}

// DeletePoll deletes a poll by ID from Firestore
func (s *Store) DeletePoll(id string) error {
	ctx := context.Background()
	docRef := s.client.Collection("polls").Doc(id)

	doc, err := docRef.Get(ctx)
	if err != nil || !doc.Exists() {
		return errors.New("poll not found")
	}

	_, err = docRef.Delete(ctx)
	return err
}
