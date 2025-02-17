package main

import (
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/sqids/sqids-go"
	"github.com/stretchr/testify/assert"
)

func setupTestApi(t humatest.TB) humatest.TestAPI {
	app := App{
		db: setupDB(os.Getenv("DB_URL")),
	}
	s, _ := sqids.New(sqids.Options{
		MinLength: 9,
	})
	app.s = s

	_, api := humatest.New(t)
	app.addRoutes(api)
	return api
}

func getLeaderboard(api humatest.TestAPI, leaderboard_id string) (LeaderboardResponseBody, *httptest.ResponseRecorder) {
	getResp := api.Get(fmt.Sprintf("/leaderboard/%s", leaderboard_id))
	var lResp LeaderboardResponseBody
	json.Unmarshal(getResp.Body.Bytes(), &lResp)
	return lResp, getResp

}

func TestGetBadIDEmpty(t *testing.T) {
	api := setupTestApi(t)
	badID := ""
	_, getResp := getLeaderboard(api, badID)
	assert.Equal(t, 404, getResp.Code)
}

func TestGetBadIDTooShort(t *testing.T) {
	api := setupTestApi(t)
	badID := "123"
	_, getResp := getLeaderboard(api, badID)
	assert.Equal(t, 422, getResp.Code)
}

func TestGetBadIDTooLong(t *testing.T) {
	api := setupTestApi(t)
	badID := "iaersntaoirseoiaerstoieanrt"
	_, getResp := getLeaderboard(api, badID)
	assert.Equal(t, 422, getResp.Code)
}
func TestAddScores(t *testing.T) {
	api := setupTestApi(t)

	resp := api.Post("/leaderboard", map[string]any{
		"name": "test name",
	})
	assert.Equal(t, 200, resp.Code)
	var newResp NewLeaderboardResponseBody
	json.Unmarshal(resp.Body.Bytes(), &newResp)
	fmt.Println("newResp", newResp)
	assert.Equal(t, "test name", newResp.Name)

	id := newResp.Id

	if lResp, getResp := getLeaderboard(api, id); assert.Equal(t, 200, getResp.Code) {
		assert.Zero(t, len(lResp.Scores))
	}

	postResp := api.Post(
		fmt.Sprintf("/leaderboard/%s/score", id),
		map[string]any{
			"user":  "test",
			"score": 9,
		})

	assert.Equal(t, 200, postResp.Code)

	if lResp, getResp := getLeaderboard(api, id); assert.Equal(t, 200, getResp.Code) {
		assert.Equal(t, 1, len(lResp.Scores))
		assert.Equal(t, 9, lResp.Scores[0].Score)
	}

	postResp2 := api.Post(
		fmt.Sprintf("/leaderboard/%s/score", id),
		map[string]any{
			"user":  "test user 2",
			"score": 10,
		})

	assert.Equal(t, 200, postResp2.Code)

	if lResp, getResp := getLeaderboard(api, id); assert.Equal(t, 200, getResp.Code) {
		assert.Equal(t, 10, lResp.Scores[0].Score)
		assert.Equal(t, 9, lResp.Scores[1].Score)
	}

	postResp3 := api.Post(
		fmt.Sprintf("/leaderboard/%s/score", id),
		map[string]any{
			"user":  "test user 2",
			"score": 11,
		})
	assert.Equal(t, 200, postResp3.Code)

	if lResp, getResp := getLeaderboard(api, id); assert.Equal(t, 200, getResp.Code) {
		assert.Equal(t, 2, len(lResp.Scores))
		assert.Equal(t, 11, lResp.Scores[0].Score)
		assert.Equal(t, 9, lResp.Scores[1].Score)
	}
	postResp4 := api.Post(
		fmt.Sprintf("/leaderboard/%s/score", id),
		map[string]any{
			"user":  "test user 3",
			"score": 11,
		})
	assert.Equal(t, 200, postResp4.Code)

	if lResp, getResp := getLeaderboard(api, id); assert.Equal(t, 200, getResp.Code) {
		assert.Equal(t, 3, len(lResp.Scores))
		assert.Equal(t, 11, lResp.Scores[0].Score)
		assert.Equal(t, 11, lResp.Scores[1].Score)
		assert.Equal(t, 9, lResp.Scores[2].Score)
	}
}

func BenchmarkGetLeaderboard(b *testing.B) {
	api := setupTestApi(b)

	resp := api.Post("/leaderboard", map[string]any{
		"name": "test leaderboard",
	})

	var newResp NewLeaderboardResponseBody
	json.Unmarshal(resp.Body.Bytes(), newResp)
	s, _ := sqids.New()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {

		id, _ := s.Encode([]uint64{uint64(n)})

		api.Post(
			fmt.Sprintf("/leaderboard/%s/score", id),
			map[string]any{
				"user":  id,
				"score": n,
			})

		getLeaderboard(api, newResp.Id)

	}
}
