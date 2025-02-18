package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"cloud.google.com/go/logging"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/danielgtaylor/huma/v2/humacli"
	"github.com/go-chi/chi/v5"
	"github.com/spf13/cobra"
	"github.com/sqids/sqids-go"
)

type CloudConfigs struct {
	Substitutions struct {
		ServiceName    string `yaml:"_SERVICE_NAME"`
		Region         string `yaml:"_DEPLOY_REGION"`
		RepositoryName string `yaml:"_REPOSITORY"`
	} `yaml:"substitutions"`
}

var VERSION string
var CLI = ""
var DOCS = ""
var id_length = 9

func OpenAPIGenConfig() huma.Config {
	config := huma.DefaultConfig("leaderapi", VERSION)
	url := "https://api.topktoday.dev"
	config.Extensions = map[string]any{
		"host": url,
	}
	config.Servers = []*huma.Server{
		{URL: url},
	}
	if !(len(DOCS) > 0) && !(len(CLI) > 0) {
		config.DocsPath = ""
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
	huma.Get(api, "/health", app.healthCheck)
	huma.Get(api, "/leaderboard/{id}", app.getLeaderboard)
	huma.Get(api, "/leaderboard/{id}", app.getLeaderboard)
	huma.Get(api, "/leaderboard/{id}/name", app.getLeaderboardName)
	huma.Post(api, "/leaderboard/{id}/score", app.postNewScore)
	huma.Post(api, "/leaderboard", app.postNewLeaderboard)
	app.api = api
}

type Options struct {
	Debug bool   `doc:"Enable debug logging"`
	Host  string `doc:"Hostname to listen on."`
	Port  int    `doc:"Port to listen on." short:"p" default:"8888"`
}

func main() {
	log.Printf("app version: %s", VERSION)

	port, db_url := os.Getenv("PORT"), os.Getenv("DB_URL")
	app := App{
		log: &logging.Logger{},
	}
	if s, err := sqids.New(sqids.Options{MinLength: uint8(id_length)}); err != nil {
		log.Fatal(err)
	} else {
		app.s = s
	}
	r := chi.NewMux()
	api := humachi.New(r, OpenAPIGenConfig())
	app.addRoutes(api)

	if len(CLI) > 0 {
		cli := humacli.New(func(hooks humacli.Hooks, opts *Options) {
			fmt.Printf("I was run with debug:%v host:%v port%v\n",
				opts.Debug, opts.Host, opts.Port)
			server := http.Server{
				Addr:    fmt.Sprintf(":%d", opts.Port),
				Handler: r,
			}

			hooks.OnStart(func() {
				// Start your server here
				server.ListenAndServe()
			})

			hooks.OnStop(func() {
				// Gracefully shutdown your server here
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				server.Shutdown(ctx)
			})
		})
		cli.Root().AddCommand(&cobra.Command{
			Use:   "openapi",
			Short: "Print the OpenAPI spec",
			Run: func(cmd *cobra.Command, args []string) {
				b, err := app.api.OpenAPI().YAML()
				if err != nil {
					panic(err)
				}
				fmt.Println(string(b))
			}})
		cli.Run()
		return
	} else {
		if port == "" {
			port = "8080"
			log.Printf("defaulting to port %s", port)
		}
		app.db = setupDB(db_url)
		defer app.db.Close()
	}
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatal(err)
	}
}
