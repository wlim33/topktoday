//go:build integration
// +build integration

package main

import (
	"os"
	"testing"

	"github.com/danielgtaylor/huma/v2/humatest"
)

func setupTestApi(t *testing.T) humatest.TestAPI {

	ls_test_ids := []int{5173419, 5173429, 5173447, 5173457, 5173474}
	ls_subscription_ids := []int{5173419, 5173429, 5173447, 5173457, 5173474}
	app := App{
		st: setupDB(t.Context(), os.Getenv("DB_URL")),
	}
	app.parser = NewParser()
	if err := app.st.createTestUser(t.Context(), "testid", "admin", "admin@admin.admin", false, CustomerInfo{id: ls_test_ids[0], subscription_id: ls_subscription_ids[0]}); err != nil {
		t.Fatal(err)
	}
	app.st.createTestUser(t.Context(), "testid2", "player2", "admin2@admin.admin", false, CustomerInfo{id: ls_test_ids[1], subscription_id: ls_subscription_ids[1]})
	app.st.createTestUser(t.Context(), "testid3", "player3", "admin3@admin.admin", false, CustomerInfo{id: ls_test_ids[2], subscription_id: ls_subscription_ids[2]})
	app.st.createTestUser(t.Context(), "anon", "Anonymous", "s@anonymous.anonymous", true, CustomerInfo{id: ls_test_ids[3], subscription_id: ls_subscription_ids[3]})
	app.st.createTestUser(t.Context(), "anon2", "Anonymous2", "s2@anonymous.anonymous", true, CustomerInfo{id: ls_test_ids[4], subscription_id: ls_subscription_ids[4]})
	_, api := humatest.New(t)
	app.addRoutes(api)
	return api
}
