package main

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

type ErrResponse struct {
	Err            error `json:"-"` // low-level runtime error
	HTTPStatusCode int   `json:"-"` // http response status code

	StatusText string `json:"status"`          // user-level status message
	AppCode    int64  `json:"code,omitempty"`  // application-specific error code
	ErrorText  string `json:"error,omitempty"` // application-level error message, for debugging
}

func (e *ErrResponse) Render(w http.ResponseWriter, r *http.Request) error {
	render.Status(r, e.HTTPStatusCode)
	return nil
}
func ErrRender(err error) render.Renderer {
	return &ErrResponse{
		Err:            err,
		HTTPStatusCode: 422,
		StatusText:     "Error rendering response.",
		ErrorText:      err.Error(),
	}
}

type UpdateScoreRequest struct {
	User  string `json:"user"`
	Score int    `json:"score"`
}

type NewLeaderboardRequest struct {
	DisplayName string `json:"name"`
}
type MessageResponse struct {
	Message string `json:"message"`
}
type ErrorResponse struct {
	Error error `json:"error"`
}

type NewLeaderboardResponse struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

type LeaderboardResponse struct {
	Scores []Entry `json:"scores"`
}

func (rd *LeaderboardResponse) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}
func (rd *NewLeaderboardResponse) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

const ContextLeaderboardIdKey string = "id"
const ContextUserKey string = "user"
const ContextScoreKey string = "score"
const ContextDisplayNameKey string = "display_name"

func UpdateScoreCtx(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var reqBody UpdateScoreRequest
		err := json.NewDecoder(r.Body).Decode(&reqBody)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		ctx := context.WithValue(r.Context(), ContextUserKey, reqBody.User)
		ctx = context.WithValue(ctx, ContextScoreKey, reqBody.Score)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
func (app *App) LeaderboardIDCtx(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		url_param := chi.URLParam(r, "id")
		leaderboard_id := app.s.Decode(url_param)[0]
		ctx := context.WithValue(r.Context(), ContextLeaderboardIdKey, leaderboard_id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func LeaderboardNameCtx(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var reqBody NewLeaderboardRequest
		err := json.NewDecoder(r.Body).Decode(&reqBody)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		ctx := context.WithValue(r.Context(), ContextDisplayNameKey, reqBody.DisplayName)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (app *App) postNewLeaderboard(w http.ResponseWriter, r *http.Request) {
	display_name := r.Context().Value(ContextDisplayNameKey).(string)

	leaderboard_id, name, db_err := app.NewLeaderBoard(display_name)
	if db_err != nil {
		render.Render(w, r, ErrRender(db_err))
		return
	}

	display_id, _ := app.s.Encode([]uint64{leaderboard_id})
	resp := NewLeaderboardResponse{
		Id:   display_id,
		Name: name,
	}
	if err := render.Render(w, r, &resp); err != nil {
		render.Render(w, r, ErrRender(err))
		return
	}
}

func (app *App) postNewScore(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	leaderboard_id := ctx.Value(ContextLeaderboardIdKey).(uint64)
	user := ctx.Value(ContextUserKey).(string)
	new_score := ctx.Value(ContextScoreKey).(int)
	if db_err := app.UpdateScore(leaderboard_id, user, new_score); db_err != nil {
		render.Render(w, r, ErrRender(db_err))
		return
	}
	render.Status(r, http.StatusOK)
}

func (app *App) getLeaderboard(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	leaderboard_id := ctx.Value(ContextLeaderboardIdKey).(uint64)

	scores, db_err := app.GetLeaderboard(leaderboard_id)
	if db_err != nil {
		render.Render(w, r, ErrRender(db_err))
		return
	}

	resp := LeaderboardResponse{
		scores,
	}
	if err := render.Render(w, r, &resp); err != nil {
		render.Render(w, r, ErrRender(err))
		return
	}
}
