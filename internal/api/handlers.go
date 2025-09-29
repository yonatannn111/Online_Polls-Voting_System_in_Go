package api

import (
	"context"
	"encoding/json"
	"math/rand"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/yonatannn111/Online_Polls-Voting_System_in_Go/internal/models"
	"github.com/yonatannn111/Online_Polls-Voting_System_in_Go/internal/storage"
	"github.com/yonatannn111/Online_Polls-Voting_System_in_Go/internal/utils"
)

type App struct {
	Store *storage.Store
}

func (a *App) Firestore(ctx context.Context) (any, error) {
	panic("unimplemented")
}

// ✅ CreatePollHandler handles poll creation
func (a *App) CreatePollHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Question string   `json:"question"`
		Options  []string `json:"options"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.JSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	if req.Question == "" || len(req.Options) < 2 {
		utils.JSON(w, http.StatusBadRequest, map[string]string{"error": "question and at least 2 options required"})
		return
	}

	// Generate unique ID
	rand.Seed(time.Now().UnixNano())
	id := generateID(8)

	votes := make(map[string]int)
	for _, opt := range req.Options {
		votes[opt] = 0
	}

	poll := &models.Poll{
		ID:       id,
		Question: req.Question,
		Options:  req.Options,
		Votes:    votes,
	}

	// Save poll in store
	if err := a.Store.CreatePoll(poll); err != nil {
		utils.JSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
		return
	}

	utils.JSON(w, http.StatusCreated, poll)
}

// ✅ GetPollHandler returns poll details
func (a *App) GetPollHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	poll, err := a.Store.GetPoll(id)
	if err != nil {
		utils.JSON(w, http.StatusNotFound, map[string]string{"error": "poll not found"})
		return
	}
	utils.JSON(w, http.StatusOK, poll)
}

// ✅ VoteHandler allows voting
func (a *App) VoteHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req struct {
		Option string `json:"option"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.JSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	if err := a.Store.Vote(id, req.Option); err != nil {
		utils.JSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	utils.JSON(w, http.StatusOK, map[string]string{"message": "vote recorded"})
}

// ✅ ListPollsHandler returns all polls
func (a *App) ListPollsHandler(w http.ResponseWriter, r *http.Request) {
	polls := a.Store.ListPolls()
	utils.JSON(w, http.StatusOK, polls)
}

// ✅ DeletePollHandler deletes a poll
func (a *App) DeletePollHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := a.Store.DeletePoll(id); err != nil {
		utils.JSON(w, http.StatusNotFound, map[string]string{"error": "poll not found"})
		return
	}

	utils.JSON(w, http.StatusOK, map[string]string{"message": "poll deleted"})
}

// Generate unique ID
func generateID(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}
