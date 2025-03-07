//go:build integration
// +build integration

package main

import (
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/stretchr/testify/assert"
)

func getLeaderboardWithModifiedHeader(api humatest.TestAPI, leaderboard_id string, if_modified_since time.Time) (LeaderboardResponseBody, *httptest.ResponseRecorder) {
	getResp := api.Get(fmt.Sprintf("/leaderboard/%s", leaderboard_id),
		fmt.Sprintf("If-Modified-Since: %s", if_modified_since.UTC().Format(http.TimeFormat)))
	var lResp LeaderboardResponseBody
	json.Unmarshal(getResp.Body.Bytes(), &lResp)
	return lResp, getResp
}

func getLeaderboard(api humatest.TestAPI, leaderboard_id string) (LeaderboardResponseBody, *httptest.ResponseRecorder) {

	getResp := api.Get(fmt.Sprintf("/leaderboard/%s", leaderboard_id))
	var lResp LeaderboardResponseBody
	json.Unmarshal(getResp.Body.Bytes(), &lResp)
	return lResp, getResp
}

func TestLeaderboardInfo(t *testing.T) {

	api := setupTestApi(t)
	display_name := "My First Leaderboard"

	id := createBasicLeaderboard(api, t, "testid")

	getNameResp := api.Get(fmt.Sprintf("/leaderboard/%s/info", id))

	var lResp LeaderboardInfo
	json.Unmarshal(getNameResp.Body.Bytes(), &lResp)

	assert.Equal(t, display_name, lResp.Title)
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

func TestNewLeaderboard(t *testing.T) {
	api := setupTestApi(t)

	id := createBasicLeaderboard(api, t, "testid")

	if lResp, getResp := getLeaderboard(api, id); assert.Equal(t, 200, getResp.Code) {
		assert.Zero(t, len(lResp.Scores))
	}

}

func TestAddScores(t *testing.T) {
	api := setupTestApi(t)

	id := createBasicLeaderboard(api, t, "testid")

	if lResp, getResp := getLeaderboard(api, id); assert.Equal(t, 200, getResp.Code) {
		assert.Zero(t, len(lResp.Scores))
	}

	postResp := api.Post(
		fmt.Sprintf("/leaderboard/%s/submission", id),
		"UserID: testid",
		map[string]any{
			"link":  "www.youtube.com",
			"score": 9,
		})

	assert.Equal(t, 200, postResp.Code)

	if lResp, getResp := getLeaderboard(api, id); assert.Equal(t, 200, getResp.Code) {
		assert.Equal(t, 1, len(lResp.Scores))
		assert.Equal(t, 9, lResp.Scores[0].Score)
	}

	postResp2 := api.Post(
		fmt.Sprintf("/leaderboard/%s/submission", id),
		"UserID: testid2",
		map[string]any{
			"link":  "www.youtube.com",
			"score": 10,
		})

	assert.Equal(t, 200, postResp2.Code)

	if lResp, getResp := getLeaderboard(api, id); assert.Equal(t, 200, getResp.Code) {
		assert.Equal(t, 2, len(lResp.Scores))
		assert.Equal(t, 10, lResp.Scores[0].Score)
		assert.Equal(t, 9, lResp.Scores[1].Score)
	}

	var newScoreBody SubmissionResponseBody
	json.Unmarshal(postResp2.Body.Bytes(), &newScoreBody)
	postResp3 := api.Patch(
		fmt.Sprintf("/leaderboard/%s/submission/%s/score", id, newScoreBody.ID),
		"UserID: testid2",
		map[string]any{
			"link":  "www.youtube.com",
			"score": 11,
		})
	assert.Equal(t, 200, postResp3.Code)

	if lResp, getResp := getLeaderboard(api, id); assert.Equal(t, 200, getResp.Code) {
		assert.Equal(t, 2, len(lResp.Scores))
		assert.Equal(t, 11, lResp.Scores[0].Score)
		assert.Equal(t, 9, lResp.Scores[1].Score)
	}

	postResp4 := api.Post(
		fmt.Sprintf("/leaderboard/%s/submission", id),
		"UserID: testid3",
		map[string]any{
			"link":  "www.youtube.com",
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
	api := setupBenchmarkApi(b)
	l_ids := []string{}

	for range 100 {
		id := benchmarkCreateBasicLeaderboard(api, b, "testid")
		l_ids = append(l_ids, id)
	}
	for range 5000 {

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

func TestLeaderboardUpdateTrigger(t *testing.T) {
	api := setupTestApi(t)
	id := createBasicLeaderboard(api, t, "testid")
	postResp2 := api.Post(
		fmt.Sprintf("/leaderboard/%s/submission", id),
		"UserID: testid",
		map[string]any{
			"link":  "www.youtube.com",
			"score": 10,
		})

	assert.Equal(t, 200, postResp2.Code)

	if _, getResp := getLeaderboardWithModifiedHeader(api, id, time.Now().AddDate(0, 1, 0)); assert.Equal(t, http.StatusNotModified, getResp.Code) {
		assert.NotEmpty(t, getResp.Header().Get("Last-Modified"))
	}
}
