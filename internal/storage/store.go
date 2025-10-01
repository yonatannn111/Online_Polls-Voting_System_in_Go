package storage

import (
	"context"
	"errors"
	"fmt"
	"log"

	"cloud.google.com/go/firestore"
	"github.com/yonatannn111/Online_Polls-Voting_System_in_Go/internal/models"
	"google.golang.org/api/iterator"
)

type Store struct {
	client *firestore.Client
	ctx    context.Context
}

// NewStore initializes Firestore storage
func NewStore(projectID string) *Store {
	ctx := context.Background()
	client, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		log.Fatalf("Failed to create Firestore client: %v", err)
	}
	return &Store{client: client, ctx: ctx}
}

// CreatePoll adds a new poll
func (s *Store) CreatePoll(poll *models.Poll) error {
	// check if poll already exists
	docRef := s.client.Collection("polls").Doc(poll.ID)
	doc, err := docRef.Get(s.ctx)
	if err == nil && doc.Exists() {
		return fmt.Errorf("poll with ID %s already exists", poll.ID)
	}

	_, err = docRef.Set(s.ctx, map[string]interface{}{
		"question": poll.Question,
		"options":  poll.Options,
		"votes":    poll.Votes,
	})
	if err != nil {
		return fmt.Errorf("failed to create poll: %v", err)
	}
	return nil
}

// GetPoll returns a poll by ID
func (s *Store) GetPoll(id string) (*models.Poll, error) {
	doc, err := s.client.Collection("polls").Doc(id).Get(s.ctx)
	if err != nil {
		return nil, errors.New("poll not found")
	}

	var poll models.Poll
	if err := doc.DataTo(&poll); err != nil {
		return nil, fmt.Errorf("failed to parse poll: %v", err)
	}
	poll.ID = id
	return &poll, nil
}

// Vote adds a vote to a poll
func (s *Store) Vote(id string, option string) error {
	docRef := s.client.Collection("polls").Doc(id)

	return s.client.RunTransaction(s.ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		doc, err := tx.Get(docRef)
		if err != nil {
			return errors.New("poll not found")
		}

		data := doc.Data()
		votes := data["votes"].(map[string]interface{})

		if _, exists := votes[option]; !exists {
			return errors.New("invalid option")
		}

		votes[option] = votes[option].(int64) + 1

		return tx.Set(docRef, map[string]interface{}{
			"question": data["question"],
			"options":  data["options"],
			"votes":    votes,
		})
	})
}

// ListPolls returns all polls
func (s *Store) ListPolls() ([]*models.Poll, error) {
	iter := s.client.Collection("polls").Documents(s.ctx)
	var polls []*models.Poll

	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to fetch polls: %v", err)
		}

		var poll models.Poll
		if err := doc.DataTo(&poll); err != nil {
			continue
		}
		poll.ID = doc.Ref.ID
		polls = append(polls, &poll)
	}
	return polls, nil
}

// DeletePoll deletes a poll by ID
func (s *Store) DeletePoll(id string) error {
	_, err := s.client.Collection("polls").Doc(id).Delete(s.ctx)
	if err != nil {
		return errors.New("poll not found or failed to delete")
	}
	return nil
}
