package api

import (
	"github.com/go-chi/chi/v5"
	"net/http"
)

func (a *App) Router() http.Handler {
	r := chi.NewRouter()

	r.Post("/polls", a.CreatePollHandler)
	r.Get("/polls", a.ListPollsHandler)
	r.Get("/polls/{id}", a.GetPollHandler)
	r.Post("/polls/{id}/vote", a.VoteHandler)

	return r
}
