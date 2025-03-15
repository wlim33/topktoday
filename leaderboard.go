package main

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
	"hash"
	"log"
	"net/http"
	"os"
	"time"

	"cloud.google.com/go/logging"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/danielgtaylor/huma/v2/humacli"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/gofrs/uuid/v5"
	lru "github.com/hashicorp/golang-lru/v2"
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

type IDParser struct {
	submissions  *sqids.Sqids
	leaderboards *sqids.Sqids
}

func OpenAPIGenConfig() huma.Config {
	config := huma.DefaultConfig("leaderapi", VERSION)
	url := "https://api.topktoday.dev"
	config.Extensions = map[string]any{
		"host": url,
	}
	config.Servers = []*huma.Server{
		{URL: url},
	}

	return config
}

type App struct {
	*http.Server
	projectID   string
	log         *logging.Logger
	st          DB
	api         huma.API
	webhookHash hash.Hash
	lsApiKey    string
	cache       *lru.TwoQueueCache[uuid.UUID, *LeaderboardResponse]
}

func (app *App) addRoutes(api huma.API) {
	huma.Get(api, "/health", app.healthCheck)

	// Leaderboards
	huma.Register(api, huma.Operation{
		OperationID: "new-leaderboard",
		Method:      http.MethodPost,
		Path:        "/leaderboard",
		Middlewares: huma.Middlewares{app.CustomerMiddleware},
	}, app.postNewLeaderboard)

	huma.Register(api, huma.Operation{
		OperationID: "get-leaderboard",
		Method:      http.MethodGet,
		Path:        "/leaderboard/{leaderboard_id}",
	}, app.getLeaderboard)
	huma.Get(api, "/leaderboard/{leaderboard_id}/info", app.getLeaderboardInfo)
	huma.Get(api, "/leaderboard/{leaderboard_id}/verifiers", app.getLeaderboardVerifiers)

	// Submissions
	huma.Post(api, "/leaderboard/{leaderboard_id}/submission", app.postNewScore)
	huma.Get(api, "/leaderboard/{leaderboard_id}/submission/{submission_id}", app.getSubmission)
	huma.Get(api, "/leaderboard/{leaderboard_id}/submission/{submission_id}/history", app.GetSubmissionHistory)
	// huma.Patch(api, "/leaderboard/{leaderboard_id}/submission/{submission_id}/score", app.updateSubmission)
	huma.Patch(api, "/leaderboard/{leaderboard_id}/submission/{submission_id}/verify", app.VerifyScore)
	huma.Post(api, "/leaderboard/{leaderboard_id}/submission/{submission_id}/comment", app.AddSubmissionComment)

	// Accounts
	huma.Get(api, "/account/{user_id}/leaderboards", app.getAccountLeaderboards)
	huma.Get(api, "/account/{user_id}/submissions", app.getAccountSubmissions)
	huma.Post(api, "/account/link_anonymous", app.linkAnonymousAccount)

	// Webhooks

	huma.Register(api, huma.Operation{
		OperationID:        "webhook",
		Method:             http.MethodPost,
		Path:               "/webhooks/lemon_squeezy",
		SkipValidateParams: true,
		Hidden:             true,
	}, app.lemonPost)

	app.api = api
}

type Options struct {
	Debug bool   `doc:"Enable debug logging"`
	Host  string `doc:"Hostname to listen on."`
	Port  int    `doc:"Port to listen on." short:"p" default:"8888"`
}

func main() {
	log.Printf("App version: %s", VERSION)
	port, db_url, ls_secret, api_key := os.Getenv("PORT"), os.Getenv("DB_URL"), os.Getenv("LS_SECRET"), os.Getenv("PAYMENT_API_KEY")
	app := App{
		log:         &logging.Logger{},
		webhookHash: hmac.New(sha256.New, []byte(ls_secret)),
		lsApiKey:    api_key,
		cache:       initCache(),
	}

	r := chi.NewMux()

	r.Use(cors.Handler(cors.Options{
		// AllowedOrigins:   []string{"https://foo.com"}, // Use this to allow specific origin hosts
		AllowedOrigins: []string{"https://*", "http://*"},
		// AllowOriginFunc:  func(r *http.Request, origin string) bool { return true },
		AllowedMethods:   []string{"GET", "PATCH", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token", "X-Signature", "X-Event-Name"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300, // Maximum value not ignored by any of major browsers
	}))

	api := humachi.New(r, OpenAPIGenConfig())
	app.addRoutes(api)

	if port == "" {
		port = "8080"
		log.Printf("Defaulting to port %s", port)
	}
	cli := humacli.New(func(hooks humacli.Hooks, opts *Options) {
		server := http.Server{
			Addr:    fmt.Sprintf(":%d", opts.Port),
			Handler: r,
		}

		hooks.OnStart(func() {
			// Start your server here
			app.st = NewDBConn(context.Background(), db_url)

			if err := http.ListenAndServe(":"+port, r); err != nil {
				log.Fatal(err)
			}
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
}
