package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"

	"cloud.google.com/go/logging"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	"github.com/sqids/sqids-go"
	"gopkg.in/yaml.v3"
)

type CloudConfigs struct {
	Substitutions struct {
		ServiceName    string `yaml:"_SERVICE_NAME"`
		Region         string `yaml:"_DEPLOY_REGION"`
		RepositoryName string `yaml:"_REPOSITORY"`
	} `yaml:"substitutions"`
}

func OpenAPIGenConfig() huma.Config {
	var c CloudConfigs
	file, _ := os.ReadFile("cloudbuild.yaml")
	yaml.Unmarshal(file, &c)

	config := huma.DefaultConfig("leaderapi", "0.0.1")
	url := "www.example.com"
	config.Extensions = map[string]any{
		"host":           fmt.Sprintf("%s.appspot.com", c.Substitutions.RepositoryName),
		"x-google-allow": "all",
		"x-google-backend": map[string]string{
			"address": url,
		},
	}
	config.Servers = []*huma.Server{
		{URL: url},
	}

	return config
}

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
	api := humachi.New(r, OpenAPIGenConfig())
	app.addRoutes(api)

	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatal(err)
	}
}
