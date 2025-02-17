package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"

	"cloud.google.com/go/logging"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	"github.com/sqids/sqids-go"
)

type App struct {
	*http.Server
	projectID string
	log       *logging.Logger
	db        *sql.DB
	s         *sqids.Sqids
	api       huma.API
}

func (app *App) addRoutes(api huma.API) {
	huma.Get(api, "/leaderboard/{id}", app.getLeaderboard)
	huma.Get(api, "/leaderboard/{id}/name", app.getLeaderboardName)
	huma.Post(api, "/leaderboard/{id}/score", app.postNewScore)
	huma.Post(api, "/leaderboard", app.postNewLeaderboard)
	app.api = api
}

func main() {
	port, db_url := os.Getenv("PORT"), os.Getenv("DB_URL")
	if port == "" {
		port = "8080"
		log.Printf("defaulting to port %s", port)
	}
	app := App{
		db:  setupDB(db_url),
		log: &logging.Logger{},
	}
	defer app.db.Close()

	if s, err := sqids.New(sqids.Options{MinLength: 9}); err != nil {
		log.Fatal(err)
	} else {
		app.s = s
	}

	r := chi.NewMux()
	api := humachi.New(r, huma.DefaultConfig("TopKTodayLeaderboard", "0.0.1"))
	app.addRoutes(api)

	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatal(err)
	}
}
