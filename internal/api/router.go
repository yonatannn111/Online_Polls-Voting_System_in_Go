package api

import (
	"github.com/go-chi/chi/v5"
	"net/http"
)

func (a *App) Router() http.Handler {
	r := chi.NewRouter()

	// ✅ Poll routes
	r.Post("/polls", a.CreatePollHandler)     // Create a new poll
	r.Get("/polls", a.ListPollsHandler)       // List all polls
	r.Get("/polls/{id}", a.GetPollHandler)    // Get single poll by ID
	r.Post("/polls/{id}/vote", a.VoteHandler) // Vote on a poll
	r.Delete("/polls/{id}", a.DeletePollHandler) // Delete a poll

	return r
}
