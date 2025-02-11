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
	"github.com/sqids/sqids-go"
)

type App struct {
	*http.Server
	projectID string
	log       *logging.Logger
	db        *sql.DB
	s         *sqids.Sqids
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
			r.Use(app.LeaderboardIDCtx)
			r.Get("/", app.getLeaderboard)
			r.With(UpdateScoreCtx).Post("/score", app.postNewScore)
		})
	})
	return r
}

func main() {

	port := os.Getenv("PORT")
	db_url := os.Getenv("DB_URL")

	app := App{
		db:  setupDB(db_url),
		log: &logging.Logger{},
	}

	s, err := sqids.New(sqids.Options{
		MinLength: 9,
	})

	if err != nil {
		log.Fatal(err)
	}
	app.s = s

	defer app.db.Close()
	if port == "" {
		port = "8080"
		log.Printf("defaulting to port %s", port)
	}

	r := app.setupRouter()
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatal(err)
	}
}
