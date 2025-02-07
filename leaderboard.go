package main

import (
	"context"
	"encoding/json"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/google/uuid"
	"log"
	"net/http"
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

type MessageResponse struct {
	Message string `json:"message"`
}
type ErrorResponse struct {
	Error error `json:"error"`
}

type NewLeaderboardResponse struct {
	Id uuid.UUID `json:"id"`
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
func LeaderboardIDCtx(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		leaderboard_id, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		ctx := context.WithValue(r.Context(), ContextLeaderboardIdKey, leaderboard_id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (sw *StorageWrapper) postNewLeaderboard(w http.ResponseWriter, r *http.Request) {
	leaderboard_id, db_err := sw.NewLeaderBoard()
	if db_err != nil {
		render.Render(w, r, ErrRender(db_err))
		return
	}
	resp := NewLeaderboardResponse{
		leaderboard_id,
	}
	if err := render.Render(w, r, &resp); err != nil {
		render.Render(w, r, ErrRender(err))
		return
	}
}

func (sw *StorageWrapper) postNewScore(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	leaderboard_id := ctx.Value(ContextLeaderboardIdKey).(uuid.UUID)
	user := ctx.Value(ContextUserKey).(string)
	new_score := ctx.Value(ContextScoreKey).(int)
	if db_err := sw.UpdateScore(leaderboard_id, user, new_score); db_err != nil {
		render.Render(w, r, ErrRender(db_err))
		return
	}
	render.Status(r, http.StatusOK)
}

func (sw *StorageWrapper) getLeaderboard(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	leaderboard_id := ctx.Value(ContextLeaderboardIdKey).(uuid.UUID)

	scores, db_err := sw.GetLeaderboard(leaderboard_id)
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

func setupRouter(storage *StorageWrapper) *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)
	r.Use(middleware.URLFormat)
	r.Use(render.SetContentType(render.ContentTypeJSON))
	r.Get("/ping", func(w http.ResponseWriter, r *http.Request) {
		resp := MessageResponse{Message: "pong"}
		json.NewEncoder(w).Encode(&resp)
	})
	r.Route("/leaderboard", func(r chi.Router) {
		r.Post("/", storage.postNewLeaderboard)
		r.Route("/{id}", func(r chi.Router) {
			r.Use(LeaderboardIDCtx)
			r.Get("/", storage.getLeaderboard)
			r.With(UpdateScoreCtx).Post("/score", storage.postNewScore)
		})
	})
	return r

}

func main() {
	storage, err := NewStorage()
	if err != nil {
		log.Fatal(err)
	}

	defer storage.Close()

	r := setupRouter(&storage)
	http.ListenAndServe(":3000", r)
}
