//go:build integration
// +build integration

package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"os"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/stretchr/testify/assert"
)

func setupBenchmarkApi(t *testing.B) humatest.TestAPI {
	ls_test_ids := []int{5173419, 5173429, 5173447, 5173457, 5173474}
	ls_subscription_ids := []int{5173419, 5173429, 5173447, 5173457, 5173474}
	app := App{
		st:          setupDB(t.Context(), os.Getenv("DB_URL")),
		webhookHash: hmac.New(sha256.New, []byte("test_key")),
		cache:       initCache(),
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

func setupTestApi(t *testing.T) humatest.TestAPI {

	ls_test_ids := []int{5173419, 5173429, 5173447, 5173457, 5173474}
	ls_subscription_ids := []int{5173419, 5173429, 5173447, 5173457, 5173474}
	app := App{
		st:          setupDB(t.Context(), os.Getenv("DB_URL")),
		webhookHash: hmac.New(sha256.New, []byte("test_key")),
		cache:       initCache(),
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

func setupTestWebhookAPI(t *testing.T, signing_key string) humatest.TestAPI {

	ls_test_ids := []int{5173419, 5173429, 5173447, 5173457, 5173474}
	ls_subscription_ids := []int{5173419, 5173429, 5173447, 5173457, 5173474}
	app := App{
		st:          setupDB(t.Context(), os.Getenv("DB_URL")),
		webhookHash: hmac.New(sha256.New, []byte(signing_key)),
		cache:       initCache(),
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

func benchmarkCreateBasicLeaderboard(api humatest.TestAPI, b *testing.B, userid string) string {
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

func createBasicLeaderboard(api humatest.TestAPI, t *testing.T, userid string) string {
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
	api := setupBenchmarkApi(b)
	l_ids := []string{}

	for range l_count {
		id := benchmarkCreateBasicLeaderboard(api, b, "testid")
		l_ids = append(l_ids, id)
	}
	for range s_count {

		postResp := api.Post(
			fmt.Sprintf("/leaderboard/%s/submission", l_ids[rand.IntN(len(l_ids))]),
			"UserID: testid3",
			map[string]any{
				"link":  "www.youtube.com",
				"score": rand.IntN(500000000),
			})

		assert.Equal(b, 200, postResp.Code)
	}

	b.ResetTimer()
	for b.Loop() {
		_, getResp := getLeaderboard(api, l_ids[rand.IntN(len(l_ids))])
		assert.Equal(b, 200, getResp.Code)
	}

}
