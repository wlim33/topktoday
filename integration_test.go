//go:build integration
// +build integration

package main

import (
	"os"
	"testing"

	"github.com/danielgtaylor/huma/v2/humatest"
)

func setupTestApi(t *testing.T) humatest.TestAPI {

	app := App{
		st: setupDB(t.Context(), os.Getenv("DB_URL")),
	}
	app.parser = NewParser()

	app.st.createTestUser(t.Context(), "testid", "admin", "admin@admin.admin", false)

	app.st.createTestUser(t.Context(), "testid2", "player2", "a@admin.admin", false)
	app.st.createTestUser(t.Context(), "testid3", "player3", "s@admin.admin", false)
	app.st.createTestUser(t.Context(), "anon", "Anonymous", "s@anonymous.anonymous", true)
	_, api := humatest.New(t)
	app.addRoutes(api)
	return api
}
