//go:build integration
// +build integration

package main

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log"
	"math/rand/v2"
	"os"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/gofrs/uuid/v5"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
)

type TestContext struct {
	api   humatest.TestAPI
	users map[string]string
}

func setupTestData(ctx context.Context, signing_key string, tx pgx.Tx) (App, TestContext) {
	testCtx := TestContext{
		users: map[string]string{},
	}
	ls_test_ids := []int{5173419, 5173429, 5173447, 5173457, 5173474}
	ls_subscription_ids := []int{5173419, 5173429, 5173447, 5173457, 5173474}
	testCtx.users["admin"] = "meow-admin"
	testCtx.users["player1"] = "meow-player1"
	testCtx.users["player2"] = "meow-player2"
	testCtx.users["player3"] = "meow-player3"
	testCtx.users["Anonymous1"] = "meow-Anonymous"
	testCtx.users["Anonymous2"] = "meow-Anonymous2"

	db := DB{
		conn: tx,
	}
	if err := db.createTestUser(ctx, testCtx.users["admin"], "admin", "admin@admin.admin", false, CustomerInfo{id: ls_test_ids[0], subscription_id: ls_subscription_ids[0]}); err != nil {
		log.Fatal(err)
	}
	db.createTestUser(ctx, testCtx.users["player2"], "player2", "admin2@admin.admin", false, CustomerInfo{id: ls_test_ids[1], subscription_id: ls_subscription_ids[1]})
	db.createTestUser(ctx, testCtx.users["player3"], "player3", "admin3@admin.admin", false, CustomerInfo{id: ls_test_ids[2], subscription_id: ls_subscription_ids[2]})
	db.createTestUser(ctx, testCtx.users["Anonymous1"], "Anonymous1", "s@anonymous.anonymous", true, CustomerInfo{id: ls_test_ids[3], subscription_id: ls_subscription_ids[3]})
	db.createTestUser(ctx, testCtx.users["Anonymous2"], "Anonymous2", "s2@anonymous.anonymous", true, CustomerInfo{id: ls_test_ids[4], subscription_id: ls_subscription_ids[4]})

	app := App{
		st:          db,
		webhookHash: hmac.New(sha256.New, []byte(signing_key)),
		cache:       initCache(),
	}
	return app, testCtx
}

func setupBenchmarkApi(t *testing.B) TestContext {
	db := NewDBConn(t.Context(), os.Getenv("DB_URL"))

	test_tx, err := db.conn.Begin(t.Context())
	if err != nil {
		log.Fatal("Failed to setup transaction:", err)
	}

	app, testData := setupTestData(t.Context(), "aoiers", test_tx)
	_, api := humatest.New(t)
	app.addRoutes(api)
	testData.api = api
	return testData
}

func benchmarkCreateBasicLeaderboard(api humatest.TestAPI, b *testing.B, userid string) uuid.UUID {
	resp := api.Post("/leaderboard",
		fmt.Sprintf("UserID: %s", userid),
		map[string]any{
			"title":                "My First Leaderboard",
			"duration":             "24:01:00",
			"highest_first":        true,
			"is_time":              true,
			"start":                time.Now().Format(time.RFC3339),
			"multiple_submissions": true,
		})
	var newResp NewLeaderboardResponseBody
	json.Unmarshal(resp.Body.Bytes(), &newResp)

	return newResp.Id
}

func createDefaultLeaderboard(t *testing.T, api humatest.TestAPI, userid string) uuid.UUID {
	t.Helper()
	resp := api.Post("/leaderboard",
		fmt.Sprintf("UserID: %s", userid),
		map[string]any{
			"title":                "My Default Leaderboard",
			"highest_first":        true,
			"is_time":              true,
			"multiple_submissions": true,
		})
	assert.Equal(t, 200, resp.Code)
	var newResp NewLeaderboardResponseBody
	json.Unmarshal(resp.Body.Bytes(), &newResp)
	return newResp.Id
}

func createBasicLeaderboard(t *testing.T, api humatest.TestAPI, userid string) uuid.UUID {
	t.Helper()

	resp := api.Post("/leaderboard",
		fmt.Sprintf("UserID: %s", userid),
		map[string]any{
			"title":                "My First Leaderboard",
			"duration":             "20:01:00",
			"highest_first":        true,
			"is_time":              true,
			"start":                time.Now().Format(time.RFC3339),
			"multiple_submissions": true,
		})
	assert.Equal(t, 200, resp.Code)
	var newResp NewLeaderboardResponseBody
	json.Unmarshal(resp.Body.Bytes(), &newResp)
	return newResp.Id
}

func Benchmark50Leaderboards100Submissions(b *testing.B) {
	benchmarkGetLeaderboard(50, 100, b)
}

func Benchmark50Leaderboards1000Submissions(b *testing.B) {
	benchmarkGetLeaderboard(50, 1000, b)
}

func Benchmark50Leaderboards10000Submissions(b *testing.B) {
	benchmarkGetLeaderboard(50, 10000, b)
}

func Benchmark10Leaderboards1000Submissions(b *testing.B) {
	benchmarkGetLeaderboard(10, 1000, b)
}

func Benchmark10Leaderboards10000Submissions(b *testing.B) {
	benchmarkGetLeaderboard(10, 10000, b)
}

func benchmarkGetLeaderboard(l_count, s_count int, b *testing.B) {
	test_ctx := setupBenchmarkApi(b)
	api := test_ctx.api
	l_ids := []uuid.UUID{}

	for range l_count {
		id := benchmarkCreateBasicLeaderboard(api, b, test_ctx.users["Anonymous"])
		if id.IsNil() {
			b.FailNow()
		}
		l_ids = append(l_ids, id)
	}
	for range s_count {

		postResp := api.Post(
			fmt.Sprintf("/leaderboard/%s/submission", l_ids[rand.IntN(len(l_ids))]),
			fmt.Sprintf("UserID: %s", test_ctx.users["player2"]),
			map[string]any{
				"link":  "www.youtube.com",
				"score": rand.IntN(500000000),
			})

		assert.Equal(b, 200, postResp.Code)
	}

	b.ResetTimer()
	for b.Loop() {
		_, getResp := getLeaderboard(b, api, l_ids[rand.IntN(len(l_ids))])
		assert.Equal(b, 200, getResp.Code)
	}

}

func WithTx(t *testing.T, f func(ctx context.Context, tx pgx.Tx)) {
	db := NewDBConn(t.Context(), os.Getenv("DB_URL"))
	test_tx, err := db.conn.Begin(t.Context())
	if err != nil {
		log.Fatal("Failed to setup transaction:", err)
	}
	f(t.Context(), test_tx)
	test_tx.Rollback(t.Context())

}

func WithAppSigningKey(t *testing.T, signing_key string, f func(ctx context.Context, api humatest.TestAPI, users map[string]string)) {
	t.Helper()
	db := NewDBConn(t.Context(), os.Getenv("DB_URL"))

	test_tx, err := db.conn.Begin(t.Context())
	if err != nil {
		log.Fatal("Failed to setup transaction:", err)
	}

	app, testData := setupTestData(t.Context(), signing_key, test_tx)

	_, api := humatest.New(t)
	app.addRoutes(api)

	f(t.Context(), api, testData.users)

	test_tx.Rollback(t.Context())

}

func WithApp(t *testing.T, f func(ctx context.Context, api humatest.TestAPI, users map[string]string)) {
	t.Helper()
	db := NewDBConn(t.Context(), os.Getenv("DB_URL"))

	test_tx, err := db.conn.Begin(t.Context())
	if err != nil {
		log.Fatal("Failed to setup transaction:", err)
	}

	app, testData := setupTestData(t.Context(), "aoiers", test_tx)

	_, api := humatest.New(t)
	app.addRoutes(api)

	f(t.Context(), api, testData.users)

	test_tx.Rollback(t.Context())

}
