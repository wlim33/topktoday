package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"

	"cloud.google.com/go/logging"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type App struct {
	*http.Server
	projectID string
	log       *logging.Logger
	db        *sql.DB
}

func (app *App) setupRouter() *chi.Mux {
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
		r.With(LeaderboardNameCtx).Post("/", app.postNewLeaderboard)
		r.Route("/{id}", func(r chi.Router) {
			r.Use(LeaderboardIDCtx)
			r.Get("/", app.getLeaderboard)
			r.With(UpdateScoreCtx).Post("/score", app.postNewScore)
		})
	})
	return r
}

func main() {
	app := App{
		db: setupDB(),
	}

	defer app.db.Close()
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
		log.Printf("defaulting to port %s", port)
	}

	r := app.setupRouter()
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatal(err)
	}
}
